#pragma comment(linker, "/ENTRY:DllMain")
#pragma comment(linker, "/SECTION:.text,ERW /MERGE:.rdata=.text /MERGE:.data=.text /MERGE:.text1=.text /SECTION:.idata,ERW")
#pragma comment(linker, "/SECTION:.Asuna,ERW /MERGE:.text=.Asuna")
#pragma comment(lib, "vfw32.lib")
#pragma comment(lib, "WINMM.lib")

#include "ml.cpp"

ML_OVERLOAD_NEW

#define METHOD_PTR(_method) PtrAdd((PVOID)NULL, _method)

typedef struct
{
    ULONG       Message;    // windows message
    ULONG       Code;       // control code or WM_NOTIFY code
    ULONG       ID;         // control ID (or 0 for windows messages)
    ULONG       LastID;     // used for entries specifying a range of control id's
    UINT_PTR    Signature;  // signature type (action) or pointer to message #
    PVOID       Function;   // routine to call (or special value)

} PP_MESSAGE_MAP_ENTRY, *PPP_MESSAGE_MAP_ENTRY;


API_POINTER(IsIconic) StubIsIconic;


BOOL NTAPI PpChooseFontW(LPCHOOSEFONTW Font)
{
    SET_FLAG(Font->Flags, 0x3000000);
    return ChooseFontW(Font);
}

PVOID NTAPI PpGetProcAddress(PVOID DllHandle, PCSTR Function)
{
    if ((ULONG_PTR)Function >= 0x10000) switch (HashAPI(Function))
    {
        case COMDLG32_ChooseFontW:
            return PpChooseFontW;
    }

    return GetRoutineAddress(DllHandle, Function);
}

#define CMD_RANGE_TRAP_CONTEXT TAG4('CRTC')

BOOL NTAPI PpIsIconic(HWND hWnd)
{
    PTEB_ACTIVE_FRAME Frame;

    Frame = FindThreadFrame(CMD_RANGE_TRAP_CONTEXT);
    if (Frame != nullptr)
        Frame->Data = (ULONG_PTR)hWnd;

    return StubIsIconic(hWnd);
}

struct PpCmdTarget
{
    READONLY_PROPERTY(HWND, m_hWnd)
    {
        return *(HWND *)PtrAdd(this, 0x20);
    }

    VOID OnCmdRange(ULONG ID)
    {

        HWND hWnd;
        int  x, y;
        RECT rcWindow, rcDesktop;

        {
            TEB_ACTIVE_FRAME Frame;

            Frame.Context = CMD_RANGE_TRAP_CONTEXT;
            Frame.Data = NULL;
            Frame.Push();

            (this->*StubOnCmdRange)(ID);

            hWnd = (HWND)Frame.Data;
        }

        if (hWnd == NULL)
            return;

        if (IsZoomed(hWnd))
            return;

        SystemParametersInfoW(SPI_GETWORKAREA, 0, &rcDesktop, 0);
        GetWindowRect(hWnd, &rcWindow);

        x = ((rcDesktop.right - rcDesktop.left) - (rcWindow.right - rcWindow.left)) >> 1;
        y = ((rcDesktop.bottom - rcDesktop.top) - (rcWindow.bottom - rcWindow.top)) >> 1;

        SetWindowPos(hWnd, HWND_NOTOPMOST, x, y, 0, 0, SWP_NOSIZE);
    }

    static TYPE_OF(&PpCmdTarget::OnCmdRange) StubOnCmdRange;
};

TYPE_OF(PpCmdTarget::StubOnCmdRange) PpCmdTarget::StubOnCmdRange = nullptr;

PPP_MESSAGE_MAP_ENTRY SearchMessageEntry(PVOID PotPlayer)
{
    PPP_MESSAGE_MAP_ENTRY MessageEntry;
    ULONG Entry[] = { WM_COMMAND, 0, 10230, 10238 };

    SEARCH_PATTERN_DATA Pattern[] =
    {
        ADD_PATTERN(Entry),
    };

    MessageEntry = (PPP_MESSAGE_MAP_ENTRY)SearchPatternSafe(Pattern, countof(Pattern), PotPlayer, FindLdrModuleByHandle(PotPlayer)->SizeOfImage);

    return MessageEntry == nullptr ? (PPP_MESSAGE_MAP_ENTRY)IMAGE_INVALID_VA : MessageEntry;
}

BOOL UnInitialize(PVOID BaseAddress)
{
    return FALSE;
}

VOID Initialize2(PLDR_MODULE PotPlayer, BOOL Force);

BOOL WINAPI PotPlayerDllMain(PVOID BaseAddress, ULONG Reason, PVOID Reserved)
{
    BOOL Success;
    PLDR_MODULE Self;
    PIMAGE_NT_HEADERS NtHeaders;

    Self = FindLdrModuleByHandle(BaseAddress);
    NtHeaders = ImageNtHeadersFast(BaseAddress);
    Self->EntryPoint = PtrAdd(BaseAddress, NtHeaders->OptionalHeader.AddressOfEntryPoint);

    Success = ((API_POINTER(PotPlayerDllMain))Self->EntryPoint)(BaseAddress, Reason, Reserved);

    Initialize2(Self, TRUE);

    return Success;
}

VOID Initialize2(PLDR_MODULE PotPlayer, BOOL Force)
{
    if (Force == FALSE && FLAG_OFF(PotPlayer->Flags, LDRP_PROCESS_ATTACH_CALLED))
    {
        PotPlayer->EntryPoint = PotPlayerDllMain;
        return;
    }

    PVOID BaseAddress = PotPlayer->DllBase;

    Mp::PATCH_MEMORY_DATA p[] =
    {
        Mp::MemoryPatchVa((ULONG64)METHOD_PTR(&PpCmdTarget::OnCmdRange), sizeof(PVOID), &SearchMessageEntry(BaseAddress)->Function),
        Mp::FunctionJumpVa(LookupExportTable(GetKernel32Handle(), KERNEL32_GetProcAddress), PpGetProcAddress),
        Mp::FunctionJumpVa(IsIconic, PpIsIconic, &StubIsIconic),
    };

    Mp::PatchMemory(p, countof(p), BaseAddress);

    *(PULONG_PTR)&PpCmdTarget::StubOnCmdRange = p[0].Memory.Backup;
}

BOOL Initialize(PVOID BaseAddress)
{
    ml::MlInitialize();
    LdrDisableThreadCalloutsForDll(BaseAddress);

    Initialize2(FindLdrModuleByName(&USTR(L"PotPlayer.dll")), FALSE);

    return TRUE;
}

BOOL WINAPI DllMain(PVOID BaseAddress, ULONG Reason, PVOID Reserved)
{
    switch (Reason)
    {
        case DLL_PROCESS_ATTACH:
            return Initialize(BaseAddress) || UnInitialize(BaseAddress);

        case DLL_PROCESS_DETACH:
            UnInitialize(BaseAddress);
            break;
    }

    return TRUE;
}