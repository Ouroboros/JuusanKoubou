#include "Hooks.h"

HRESULT
(NTAPI
*StubShowDBClickPicture)(
    PVOID   This,
    PWSTR   PicturePath,
    PRECT   ClickPosition,
    BOOL    FromHistoryIfFalse,
    PVOID   Unknown
);

HRESULT
NTAPI
ShowDBClickPicture(
    PVOID   This,
    PWSTR   PicturePath,
    PRECT   ClickPosition,
    BOOL    FromHistoryIfFalse,
    PVOID   Unknown
)
{
    SHELLEXECUTEINFOW ShellExecueInfo;

    if (GetKeyState(VK_CONTROL) < 0)
        return StubShowDBClickPicture(This, PicturePath, ClickPosition, FromHistoryIfFalse, Unknown);

    ZeroMemory(&ShellExecueInfo, sizeof(ShellExecueInfo));
    ShellExecueInfo.cbSize  = sizeof(ShellExecueInfo);
    ShellExecueInfo.lpFile  = PicturePath;
    ShellExecueInfo.lpVerb  = L"open";

    ShellExecuteExW(&ShellExecueInfo);

    return S_OK;
}

PVOID SearchAppMisc_PluginListCheck(PVOID ImageBase)
{
    static WCHAR String[] = L"PluginListCheck¡¡Begin";
    
    return SearchStringAndReverseSearchHeader(ImageBase, String, sizeof(String), 0x60);
}

PVOID SearchAppMisc_ShowPicInMultiPic(PVOID ImageBase)
{
    static WCHAR String[] = L"ShowPicInMultiPic begin";
    
    return SearchStringAndReverseSearchHeader(ImageBase, String, sizeof(String) - sizeof(WCHAR), 0x20);
}

NTSTATUS HookAppMisc(PVOID BaseAddress)
{
    /************************************************************************
      AppMisc

        L"PluginListCheck"
        mov     eax, 0x80004005

        ShowPicInMultiPic begin
    ************************************************************************/

    //Util::GF::DPI::SetDPIAdaptFlag(FALSE);

    Mp::PATCH_MEMORY_DATA Function_AppMisc[] =
    {
        Mp::FunctionJumpVa(SearchAppMisc_PluginListCheck(BaseAddress),    CheckPluginList), // addition SetTimeOut
        //Mp::FunctionJumpVa(SearchAppMisc_ShowPicInMultiPic(BaseAddress),  ShowDBClickPicture, &StubShowDBClickPicture),
    };

    return Mp::PatchMemory(Function_AppMisc, countof(Function_AppMisc), BaseAddress);
}