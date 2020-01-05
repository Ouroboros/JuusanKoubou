#pragma comment(linker, "/ENTRY:DllMain")
#pragma comment(linker, "/SECTION:.text,ERW /MERGE:.rdata=.text /MERGE:.data=.text")
#pragma comment(linker, "/SECTION:.Asuna,ERW /MERGE:.text=.Asuna")

// #pragma comment(linker, "/EXPORT:WTSFreeMemory=_QqWTSFreeMemory@4")
// #pragma comment(linker, "/EXPORT:WTSQuerySessionInformationW=_QqWTSQuerySessionInformationW@0")
// #pragma comment(linker, "/EXPORT:WTSRegisterSessionNotification=_QqWTSRegisterSessionNotification@8")
// #pragma comment(linker, "/EXPORT:WTSUnRegisterSessionNotification=_QqWTSUnRegisterSessionNotification@4")

#include "QQ2011.h"
#include "ZLibExport.h"
#include "ml.cpp"
#include <WtsApi32.h>
#include "Hooks/Hooks.h"

ML_OVERLOAD_NEW


/*++

Req AnonymousInfo %lu
AnonymousChat

--*/

TXReloginMgr* ReloginMgr;

PVOID AppUtilBase;

#if 0

API_POINTER(WTSFreeMemory)                      StubWTSFreeMemory;
API_POINTER(WTSQuerySessionInformationW)        StubWTSQuerySessionInformationW;
API_POINTER(WTSRegisterSessionNotification)     StubWTSRegisterSessionNotification;
API_POINTER(WTSUnRegisterSessionNotification)   StubWTSUnRegisterSessionNotification;

BOOL InitializeNetapi32()
{
    PVOID           module;
    NTSTATUS        Status;
    PLDR_MODULE     Self, Netapi32;
    UNICODE_STRING  SystemRoot;
    PVOID           LoaderLockCookie;

    LdrLockLoaderLock(0, nullptr, &LoaderLockCookie);
    SCOPE_EXIT
    {
        LdrUnlockLoaderLock(0, LoaderLockCookie);
    }
    SCOPE_EXIT_END;

    Self = FindLdrModuleByHandle(&__ImageBase);

    if (Self == nullptr || Self->DllBase != &__ImageBase)
        return TRUE;

    Status = Rtl::GetSystemDirectory(&SystemRoot);
    if (NT_FAILED(Status))
        return 0;

    module = Ldr::LoadDll(String(SystemRoot) + L"wtsapi32.dll");
    RtlFreeUnicodeString(&SystemRoot);

    LdrAddRefDll(LDR_ADDREF_DLL_PIN, module);

    *(PVOID *)&StubWTSFreeMemory                    = GetRoutineAddress(module, "WTSFreeMemory");
    *(PVOID *)&StubWTSQuerySessionInformationW      = GetRoutineAddress(module, "WTSQuerySessionInformationW");
    *(PVOID *)&StubWTSRegisterSessionNotification   = GetRoutineAddress(module, "WTSRegisterSessionNotification");
    *(PVOID *)&StubWTSUnRegisterSessionNotification = GetRoutineAddress(module, "WTSUnRegisterSessionNotification");

    Netapi32 = FindLdrModuleByHandle(module);

    //RemoveEntryList(&Self->InLoadOrderLinks);
    //RemoveEntryList(&Self->InMemoryOrderLinks);
    //RemoveEntryList(&Self->InInitializationOrderLinks);

    //RtlFreeHeap(CurrentPeb()->ProcessHeap, 0, Self);

    Self->DllBase       = Netapi32->DllBase;
    Self->EntryPoint    = Netapi32->EntryPoint;
    Self->SizeOfImage   = Netapi32->SizeOfImage;

    return TRUE;
}

EXTC VOID WINAPI QqWTSFreeMemory(PVOID Memory)
{
    InitializeNetapi32();
    return StubWTSFreeMemory(Memory);
}

EXTC BOOL WINAPI QqWTSQuerySessionInformationW()
{
    InitializeNetapi32();
    return ((API_POINTER(QqWTSQuerySessionInformationW))StubWTSQuerySessionInformationW)();
}

EXTC LONG WINAPI QqWTSRegisterSessionNotification(HWND hWnd, DWORD Flags)
{
    InitializeNetapi32();
    return StubWTSRegisterSessionNotification(hWnd, Flags);
}

EXTC LONG WINAPI QqWTSUnRegisterSessionNotification(HWND hWnd)
{
    InitializeNetapi32();
    return StubWTSUnRegisterSessionNotification(hWnd);
}

#endif

/************************************************************************
  init functions
************************************************************************/

PVOID
SearchStringAndReverseSearchHeader(
    PVOID       ImageBase,
    PVOID       BytesSequence,
    ULONG_PTR   SizeInBytes,
    ULONG_PTR   SearchRange
)
{
    PVOID       StringReference;
    PLDR_MODULE Module;

    Module = FindLdrModuleByHandle(ImageBase);

    StringReference = SearchStringReference(Module, BytesSequence, SizeInBytes);
    if (StringReference == nullptr)
        return IMAGE_INVALID_VA;

    StringReference = ReverseSearchFunctionHeader(PtrAdd(StringReference, 4), SearchRange);

    return StringReference == nullptr ? IMAGE_INVALID_VA : StringReference;
}

BOOL UnInitialize(PVOID BaseAddress)
{
    return FALSE;
}


typedef struct
{
    UNICODE_STRING DllName;
    NTSTATUS (*HookRoutine)(PVOID BaseAddress);

} DLL_HOOK_ENTRY, *PDLL_HOOK_ENTRY;

static DLL_HOOK_ENTRY Hooks[] =
{
    { IUSTR(L"AppUtil.dll"),        HookAppUtil },
    // { IUSTR(L"Common.dll"),         HookCommon },
    // { IUSTR(L"KernelUtil.dll"),     HookKernelUtil },

    //{ IUSTR(L"GroupApp.dll"),   HookGroupApp },

    { IUSTR(L"AppMisc.dll"),        HookAppMisc },
    // { IUSTR(L"MainFrame.dll"),      HookMainFrame },
    { IUSTR(L"ntdll.dll"),          HookNtdll },

    //{ IUSTR(L"psapi.dll"),          HookPsapi },

    //{ IUSTR(L"user32.dll"),         HookUser32 },
    // { IUSTR(L"Camera.dll"),         HookCamera },
    { IUSTR(L"IM.dll"),             HookIM },

    //{ IUSTR(L"AppFramework.dll"),   HookAppFramework },
    //{ IUSTR(L"MsgMgr.dll"),         HookMsgMgr },
    //{ IUSTR(L"GDI32.dll"),      HookGdi32 },

    {},
};

BOOL Initialize2(PVOID BaseAddress)
{
    BOOL                        QQUINSpecified;
    NTSTATUS                    Status;
    PVOID                       module;
    ULONG_PTR                   CreateThreadIAT;
    PROCESS_BASIC_INFORMATION   BasicInfo;
    PLDR_MODULE                 Self, Netapi32;

    ml::MlInitialize();
    LdrDisableThreadCalloutsForDll(BaseAddress);

    Self = FindLdrModuleByHandle(nullptr);
    if (RtlEqualUnicodeString(&Self->BaseDllName, &USTR(L"QQ.exe"), TRUE) == FALSE &&
        RtlEqualUnicodeString(&Self->BaseDllName, &USTR(L"QQ2.exe"), TRUE) == FALSE)
    {
        return TRUE;
    }

    InitializeQqFunctionTable();

    PVOID DllNotificationCookie;

    LdrRegisterDllNotification(0,
        [] (ULONG NotificationReason, PCLDR_DLL_NOTIFICATION_DATA NotificationData, PVOID Context)
        {
            PDLL_HOOK_ENTRY Entry;

            if (NotificationReason != LDR_DLL_NOTIFICATION_REASON_LOADED)
                return;

            FOR_EACH_ARRAY(Entry, Hooks)
            {
                if (RtlEqualUnicodeString(&Entry->DllName, NotificationData->Loaded.BaseDllName, TRUE) == FALSE)
                    continue;

                Entry->HookRoutine(NotificationData->Loaded.DllBase);
                break;
            }
        },
        nullptr,
        &DllNotificationCookie
    );

    PDLL_HOOK_ENTRY Entry;

    FOR_EACH_ARRAY(Entry, Hooks)
    {
        PLDR_MODULE Module;

        Module = FindLdrModuleByName(&Entry->DllName);
        if (Module != nullptr)
            Entry->HookRoutine(Module->DllBase);
    }

    return TRUE;
}

BOOL Initialize(PVOID BaseAddress)
{
    auto Apc = [] (PVOID BaseAddress, PVOID, PVOID)
    {
        Initialize2(BaseAddress);
    };

    LdrDisableThreadCalloutsForDll(BaseAddress);

    return NT_SUCCESS(NtQueueApcThread(CurrentThread, Apc, BaseAddress, 0, 0));
}

BOOL WINAPI DllMain(PVOID BaseAddress, ULONG Reason, PVOID Reserved)
{
    switch (Reason)
    {
        case DLL_PROCESS_ATTACH:
            return Initialize2(BaseAddress) || UnInitialize(BaseAddress);

        case DLL_PROCESS_DETACH:
            UnInitialize(BaseAddress);
            break;
    }

    return TRUE;
}
