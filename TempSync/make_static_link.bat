@echo off
cd/d "%~dp0"

mklink /j "AtomConfig" "D:\Dev\Atom\config"
mklink /j "SublimeText Installed Packages" "D:\Dev\SublimeText\Data\Installed Packages"
mklink /j "SublimeText Packages" "D:\Dev\SublimeText\Data\Packages"
mklink /j "VSCodeUserData" "D:\Dev\VSCode\data\user-data\User"
mklink /j "savedata\edao_savedata" "E:\Game\Falcom\ED_AO\savedata"
mklink /j "savedata\Salt and Sanctuary" "E:\Game\Steam\steamapps\common\Salt and Sanctuary\savedata"
mklink /j "Vim" "D:\Dev\Vim"
