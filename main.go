package main

import (
	"archive/zip"
	_ "embed"
	"fmt"
	"github.com/go-ole/go-ole"
	"unicode/utf16"

	"github.com/go-ole/go-ole/oleutil"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
	"golang.org/x/sys/windows"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

/*
*
cd installer 到当前目录后
安装

	gcc   https://github.com/niXman/mingw-builds-binaries/releases
	go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest

	https://github.com/upx/upx

生成ico syso文件: go generate
构建：
1: set GOARCH=amd64  |  go env -w GOARCH=amd64  go env -w CGO_ENABLED=1   go env -w GOPROXY=https://goproxy.cn
1.1: 修改版本信息 编译文件 versioninfo.json
1.2: go generate

2: go  build  -ldflags="-H windowsgui" -trimpath -o ../LuckyGameToolsInstaller.exe

note: -s：去掉符号表（symbol table）和调试信息 360会报毒
*/

var i18n = "english"

//go:embed exe/GamePower.zip
var GamePowerZip []byte

//go:embed cef/7z.zip
var z7 []byte

//go:embed cef/cef84-min.7z
var cef7Zip []byte

//go:embed exe/defaultConfig.dat.local
var configJsonDatLocal []byte

//go:embed exe/appdata.zip
var appdataZip []byte

//go:generate goversioninfo -icon=main.ico -manifest=main.manifest -64 -o main.syso

func main() {
	i18n = GetLocale()

	i18n = InitI18n(i18n)
	var mw *walk.Dialog
	var installPathEdit *walk.LineEdit
	var pb *walk.ProgressBar
	var pt *walk.PushButton
	var width = 800
	var height = 200
	var cb *walk.ComboBox

	languages := GetLocaleLangs()

	currentLangsIndex := GetLocaleCodeIndex(i18n)

	programFilesDir := os.Getenv("ProgramFiles")
	if programFilesDir == "" {
		programFilesDir = "C:\\Program Files"
	}

	// 获取所有的逻辑驱动器
	drives, err := walk.DriveNames()
	if err == nil {
		if len(drives) > 1 {
			for _, drive := range drives {
				if !strings.HasPrefix(programFilesDir, drive) {
					programFilesDir = strings.Replace(programFilesDir, programFilesDir[0:3], drive, 1)
					break
				}
			}
		}

	}

	Dialog{
		AssignTo:   &mw,
		Title:      Text("Installer"),
		MinSize:    Size{Width: width, Height: height},
		Layout:     VBox{},
		Icon:       2,
		Background: TransparentBrush{},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 3, MarginsZero: true},
				Children: []Widget{
					Label{
						Text: Text("Installer Path") + ":",
					},
					LineEdit{
						AssignTo: &installPathEdit,
						Text:     programFilesDir + "\\LuckyGameTools",
						ReadOnly: true,
					},
					PushButton{
						Text: Text("Choose Installer Path"),
						OnClicked: func() {
							dlg := new(walk.FileDialog)
							dlg.Title = Text("Choose Installer Path")
							dlg.Filter = Text("Directory") + "|*"
							dlg.FilePath = installPathEdit.Text()
							if ok, err := dlg.ShowBrowseFolder(mw); err != nil {
								walk.MsgBox(mw, Text("Error"), err.Error(), walk.MsgBoxIconError|walk.MsgBoxTopMost)
							} else if ok {
								installPathEdit.SetText(filepath.Join(dlg.FilePath, "LuckyGameTools"))
							}
						},
					},
				},
			},
			ComboBox{
				AssignTo:     &cb,
				Editable:     false,
				Model:        languages,
				CurrentIndex: currentLangsIndex, // 預設選擇第一個語言
				OnCurrentIndexChanged: func() {
					selected := cb.Text()
					i18n = GetLocaleLangsCode(selected)
				},
			},
			PushButton{
				AssignTo:    &pt,
				Text:        Text("Install"),
				ToolTipText: Text("Please Exit the LuckyGameTools Client and Steam Before Installation"),
				OnClicked: func() {
					pt.SetEnabled(false)
					go func() {
						installPath := installPathEdit.Text()
						if ret := installProgram(installPath, pb); ret != "" {
							walk.MsgBox(mw, Text("Error"), ret, walk.MsgBoxIconError|walk.MsgBoxTopMost)
							pt.SetEnabled(true)
						}
					}()
				},
			},
			ProgressBar{
				AssignTo: &pb,
				Row:      1,
				MinValue: 0,
				MaxValue: 100,
			},
		},
	}.Create(nil)

	WindowDisableChangeSize(mw.Handle())
	//win.SetWindowLong(mw.Handle(), win.GWL_EXSTYLE, win.GetWindowLong(mw.Handle(), win.GWL_EXSTYLE)|win.WS_EX_TOOLWINDOW)
	CenterWindow(mw.Handle(), width, height)

	mw.Run()
}

func runAsAdmin() bool {
	executablePath, err := os.Executable()
	if err != nil {
		return false
	}

	// 将相对路径转换为绝对路径
	absolutePath, _ := filepath.Abs(executablePath)
	if err != nil {
		return false
	}

	// 解析符号链接并获取实际路径
	realPath, err := filepath.EvalSymlinks(absolutePath)
	if err != nil {
		return false
	}
	exePath := realPath

	execute := win.ShellExecute(0,
		win.StringToBSTR("runas"),
		win.StringToBSTR(exePath),
		win.StringToBSTR(""),
		win.StringToBSTR(""),
		win.SW_SHOWNORMAL)
	if !execute {
		return false
	}

	return true
}

// CenterWindow 将窗口居中显示
func CenterWindow(w win.HWND, width, height int) {
	xScreen := win.GetSystemMetrics(win.SM_CXSCREEN)
	yScreen := win.GetSystemMetrics(win.SM_CYSCREEN)
	var centerY = (yScreen - int32(height)) / 2
	centerY = centerY - 250
	if centerY < 10 {
		centerY = 10
	}
	var centerX = (xScreen - int32(width)) / 2
	win.SetWindowPos(
		w,
		win.HWND_TOPMOST,
		centerX,
		centerY,
		int32(width),
		int32(height),
		win.SWP_FRAMECHANGED,
	)
}
func WindowDisableChangeSize(w win.HWND) {
	defaultStyle := win.GetWindowLong(w, win.GWL_STYLE) // Gets current style
	newStyle := defaultStyle &^ win.WS_THICKFRAME       // Remove WS_THICKFRAME
	win.SetWindowLong(w, win.GWL_STYLE, newStyle)
}

func installProgram(installPath string, pb *walk.ProgressBar) string {
	//os.RemoveAll(installPath)
	dir, err := os.ReadDir(installPath)

	if err == nil {
		for _, dirFile := range dir {
			if dirFile.IsDir() && dirFile.Name() == "webcache" {

			} else {
				os.Remove(filepath.Join(installPath, dirFile.Name()))
			}
		}
	}

	// 创建安装目录
	err = os.MkdirAll(installPath, os.ModePerm)

	var isSystemPath = false
	systemDrive := os.Getenv("SystemDrive")
	if systemDrive != "" {
		if strings.HasPrefix(installPath, systemDrive) {
			isSystemPath = true
		}
	}

	if err != nil {
		if runAsAdmin() {
			os.Exit(0)
		} else {
			return Text("Create Directory") + " " + installPath + " " + Text("Error") + " :" + err.Error()
		}
	}

	if IsRuning("GamePower.exe") || IsRuning("steam.exe") || IsRuning("steamwebhelper.exe") {
		return Text("Please Exit the LuckyGameTools Client and Steam Before Installation")
	}

	configJsonPath := filepath.Join(GetMyAppdataFolder(), "config.json")
	xor := Xor(configJsonDatLocal, []byte(GetHostName()))
	err = os.WriteFile(configJsonPath, xor, os.ModePerm)
	if err != nil {
		println(Text("Copy") + " " + Text("File") + " " + Text("Error") + " :" + err.Error() + "\r\n" + Text("You can try running with administrator privileges by right clicking"))
	}
	pb.SetValue(2)

	//GamePower.tmp.exe
	kitTmpExe := [17]uint8{0x47, 0x61, 0x6D, 0x65, 0x50, 0x6F, 0x77, 0x65, 0x72, 0x2E, 0x74, 0x6D, 0x70, 0x2E, 0x65, 0x78, 0x65}
	//GamePowerGui.tmp.exe
	guiTmpExe := [20]uint8{0x47, 0x61, 0x6D, 0x65, 0x50, 0x6F, 0x77, 0x65, 0x72, 0x47, 0x75, 0x69, 0x2E, 0x74, 0x6D, 0x70, 0x2E, 0x65, 0x78, 0x65}

	/*//steamPower.dll
	steamPowerDll := [14]uint8{0x73, 0x74, 0x65, 0x61, 0x6D, 0x50, 0x6F, 0x77, 0x65, 0x72, 0x2E, 0x64, 0x6C, 0x6C}

	os.Remove(filepath.Join(GetMyAppdataFolder(), string(steamPowerDll[:])))*/

	os.Remove(filepath.Join(GetMyAppdataFolder(), string(kitTmpExe[:])))
	os.Remove(filepath.Join(GetMyAppdataFolder(), string(kitTmpExe[:])) + "-")
	os.Remove(filepath.Join(GetMyAppdataFolder(), string(kitTmpExe[:])) + ".bak")

	os.Remove(filepath.Join(GetMyAppdataFolder(), string(guiTmpExe[:])))
	os.Remove(filepath.Join(GetMyAppdataFolder(), string(guiTmpExe[:])) + "-")
	os.Remove(filepath.Join(GetMyAppdataFolder(), string(guiTmpExe[:])) + ".bak")

	//fixme xor的文件会补360拦截
	appdataZipPath := filepath.Join(GetMyAppdataFolder(), "appdata.zip")
	os.WriteFile(appdataZipPath, appdataZip, os.ModePerm)
	var renameMap = map[string]string{"hid.dat.xor": "hid.dat", "hid64.dat.xor": "hid64.dat"}
	err = Unzip(appdataZipPath, GetMyAppdataFolder(), renameMap)
	pb.SetValue(6)

	/*kitExePath := filepath.Join(installPath, "GamePower.exe")
	err = os.WriteFile(kitExePath, GamePowerExe, os.ModePerm)
	if err != nil {
		if isSystemPath {
			if runAsAdmin() {
				os.Exit(0)
			}
		}
		return Text("Copy") + " " + Text("File") + " " + Text("Error") + " :" + err.Error()
	}*/
	pb.SetValue(10)

	guiExeZipPath := filepath.Join(installPath, "GamePowerGui-"+strconv.FormatUint(uint64(time.Now().Unix()), 10)+".zip")
	err = os.WriteFile(guiExeZipPath, GamePowerZip, os.ModePerm)
	if err != nil {
		if isSystemPath {
			if runAsAdmin() {
				os.Exit(0)
			}
		}
		return Text("Copy") + " " + Text("File") + " " + Text("Error") + " :" + err.Error() + "\r\n" + Text("You can try running with administrator privileges by right clicking")
	}
	pb.SetValue(20)
	{
		z7Path := filepath.Join(installPath, "7z.dat")
		err = os.WriteFile(z7Path, z7, os.ModePerm)
		if err != nil {
			return Text("Copy") + " " + Text("File") + " " + Text("Error") + " :" + err.Error() + "\r\n" + Text("You can try running with administrator privileges by right clicking")
		}
		pb.SetValue(30)

		//解压zip文件
		err = Unzip(z7Path, installPath, nil)
		if err != nil {
			return Text("Unzip") + " " + Text("File") + " " + Text("Error") + " :" + err.Error() + "\r\n" + Text("You can try running with administrator privileges by right clicking")
		} else {
			fmt.Println("Unzip 7z successful!")
		}
	}
	pb.SetValue(40)

	cefZipPath := filepath.Join(installPath, "cef.dat")
	err = os.WriteFile(cefZipPath, cef7Zip, os.ModePerm)
	if err != nil {
		return Text("Copy") + " " + Text("File") + " " + Text("Error") + " :" + err.Error() + "\r\n" + Text("You can try running with administrator privileges by right clicking")
	}
	pb.SetValue(50)

	chromeElfdllpath := filepath.Join(installPath, "chrome_elf.dll")
	if FileExists(chromeElfdllpath) {
		os.Remove(chromeElfdllpath)
	}
	pb.SetValue(60)

	//解压zip文件
	err = Un7zip(cefZipPath, installPath)
	if err != nil {
		return Text("Unzip") + " " + Text("File") + " " + Text("Error") + " (7z):" + err.Error() + "\r\n" + Text("You can try running with administrator privileges by right clicking")
	} else {
		fmt.Println("Unzip cef successful!")
	}

	//GamePowerGui.exe
	guiExe := [16]uint8{0x47, 0x61, 0x6D, 0x65, 0x50, 0x6F, 0x77, 0x65, 0x72, 0x47, 0x75, 0x69, 0x2E, 0x65, 0x78, 0x65}
	//GamePowerWin64.exe
	guiX64Exe := [18]uint8{0x47, 0x61, 0x6D, 0x65, 0x50, 0x6F, 0x77, 0x65, 0x72, 0x57, 0x69, 0x6E, 0x36, 0x34, 0x2E, 0x65, 0x78, 0x65}

	//解压guiExeZip文件
	//renameGuiExe := "GamePowerGui-" + strconv.FormatUint(uint64(time.Now().Unix()), 10) + ".exe"
	//renameGuiExe := "GamePowerGui.exe"
	renameGuiExe := string(guiX64Exe[:])

	guiExePath := filepath.Join(installPath, renameGuiExe)

	renameMap = map[string]string{string(guiExe[:]): renameGuiExe}

	err = Unzip(guiExeZipPath, installPath, renameMap)

	//创建桌标
	go func() {
		targetPath := guiExePath
		err := createShortcut("LuckyGameTools", targetPath)
		if err != nil {
			walk.MsgBox(nil, Text("Error"), Text("Create Shortcut Fail")+": "+err.Error()+"\r\n"+Text("You can try running with administrator privileges by right clicking"), walk.MsgBoxIconError|walk.MsgBoxTopMost)
			return
		}

	}()

	pb.SetValue(100)

	//运行GamePower.exe
	//exec.Command(dst, "--language="+i18n).Start()

	/*cmd := exec.Command("cmd.exe", "/C", "start", dst+" --language="+i18n)
	// 设置SysProcAttr以隐藏窗口
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	err = cmd.Start()
	if err != nil {
		println("[Error]:run GamePower.exe \r\n", err)
	}*/

	/*if IsAdmin() {
		walk.MsgBox(nil, Text("Complete"), Text("Installation complete")+" "+Text("Please start from the desktop"), walk.MsgBoxIconInformation|walk.MsgBoxTopMost)
		time.Sleep(time.Second)
	} else {*/
	exec.Command(guiExePath, "--language="+i18n, "--isInstall=true").Start()
	//}
	time.Sleep(time.Second * 2)
	os.Exit(1)

	return ""
}

func Xor(message []byte, keywords []byte) []byte {
	messageLen := len(message)
	keywordsLen := len(keywords)

	result := make([]byte, messageLen)

	for i := 0; i < messageLen; i++ {
		result[i] = message[i] ^ keywords[i%keywordsLen]
	}
	return result
}

func GetMyAppdataFolder() string {
	// 获取 APPDATA 环境变量

	appDataPath := os.Getenv("APPDATA")
	if appDataPath == "" {
		println("Error: APPDATA environment variable not set and default ./")
		appDataPath = "./"
	}

	// 在 APPDATA 路径下创建一个子目录
	myAppDataPath := filepath.Join(appDataPath, "luckygametools")
	err := os.MkdirAll(myAppDataPath, os.ModePerm)
	if err != nil {
		println("Error creating directory luckygametools:", err)
	}
	return myAppDataPath
}

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func createShortcut(appName, targetPath string) error {
	ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED|ole.COINIT_SPEED_OVER_MEMORY)
	defer ole.CoUninitialize()
	desktopPath, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return err
	}
	defer desktopPath.Release()

	ws := desktopPath.MustQueryInterface(ole.IID_IDispatch)
	defer ws.Release()

	desktopDir := filepath.Join(os.Getenv("USERPROFILE"), "Desktop")
	shortcutPath := filepath.Join(desktopDir, appName+".lnk")

	cs, err := oleutil.CallMethod(ws, "CreateShortcut", shortcutPath)
	if err != nil {
		return err
	}
	defer cs.Clear()

	// 设置快捷方式的目标路径和图标（可选）
	oleutil.PutProperty(cs.ToIDispatch(), "TargetPath", targetPath)
	//oleutil.PutProperty(cs.ToIDispatch(), "IconLocation", "C:\\Path\\To\\Your\\Icon.ico")

	// 保存快捷方式
	oleutil.CallMethod(cs.ToIDispatch(), "Save")

	fmt.Println("桌面快捷方式已创建:", shortcutPath)
	return nil
}

func IsAdmin() bool {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(&windows.SECURITY_NT_AUTHORITY, 2,
		windows.SECURITY_BUILTIN_DOMAIN_RID, windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0, &sid)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	token := windows.Token(0)
	isMember, err := token.IsMember(sid)
	if err != nil {
		return false
	}
	return isMember
}

func IsRuning(processName string) bool {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(snapshot)
	var procEntry windows.ProcessEntry32
	procEntry.Size = uint32(unsafe.Sizeof(procEntry))
	if err = windows.Process32First(snapshot, &procEntry); err != nil {
		return false
	}
	for {
		exeFileName := syscall.UTF16ToString(procEntry.ExeFile[:])

		if strings.ToLower(exeFileName) == strings.ToLower(processName) {
			return true
		}
		err = windows.Process32Next(snapshot, &procEntry)
		if err != nil {
			return false
		}
	}

	return false
}

func Un7zip(zipFile, destDir string) error {
	z7exePath := filepath.Join(destDir, "7z.exe")
	if FileExists(z7exePath) {
		cmd := exec.Command(z7exePath, "x", zipFile, "-o"+destDir, "-y", "-mmt=on", "-aos")
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow: true,
		}

		return cmd.Run()
	} else {
		cmd := exec.Command("7z", "x", zipFile, "-o"+destDir, "-y", "-mmt=on", "-aos")
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow: true,
		}

		err := cmd.Run()
		return err
	}

}

// Unzip 解压 ZIP 文件到目标目录
func Unzip(zipFile, destDir string, fileRenameMap map[string]string) error {
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}

	// 创建目标目录
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	//GamePower.exe
	gamePowerExe := [13]uint8{0x47, 0x61, 0x6D, 0x65, 0x50, 0x6F, 0x77, 0x65, 0x72, 0x2E, 0x65, 0x78, 0x65}

	// 遍历 ZIP 文件中的各个文件
	for _, file := range r.File {
		var fpath string

		// 构造文件路径
		if fileRenameMap != nil {
			rename, isOk := fileRenameMap[file.Name]
			if isOk {
				fpath = filepath.Join(destDir, rename)
			} else {
				fpath = filepath.Join(destDir, file.Name)
			}
		} else {
			fpath = filepath.Join(destDir, file.Name)
		}

		// 如果是目录，则创建目录
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
				return err
			}
			continue
		}

		// 创建文件的父目录
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		// 打开 ZIP 文件中的文件
		rc, err := file.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		if file.Name == string(gamePowerExe[:]) {
			srcFileBytes, err := io.ReadAll(rc)
			if err != nil {
				return err
			}

			gamePowerExeBakData := Xor(srcFileBytes, []byte("LuckyGameT00ls"+GetHostName()))
			gamePowerExeBakPath := filepath.Join(GetMyAppdataFolder(), "GamePower.exe.bak")
			os.WriteFile(gamePowerExeBakPath, gamePowerExeBakData, os.ModePerm)

			os.WriteFile(fpath, srcFileBytes, os.ModePerm)
			continue
		}

		// 创建目标文件
		f, err := os.Create(fpath + "-")
		if err != nil {
			return err
		}
		// 将文件内容拷贝到目标文件
		if _, err := io.Copy(f, rc); err != nil {
			return err
		}
		f.Close()

		err = os.Rename(fpath+"-", fpath)
		if err != nil {
			fmt.Printf("Failed to rename file: %v\n", err)
			return err
		}
	}

	r.Close()
	os.Remove(zipFile)
	return nil
}

/*


func Un7z(archivePath, destDir string, fileRenameMap map[string]string) error {
	defer os.Remove(archivePath)
	reader, err := sevenzip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	//GamePower.exe
	gamePowerExe := [13]uint8{0x47, 0x61, 0x6D, 0x65, 0x50, 0x6F, 0x77, 0x65, 0x72, 0x2E, 0x65, 0x78, 0x65}

	for _, file := range reader.File {
		var filePath string

		// 构造文件路径
		if fileRenameMap != nil {
			rename, isOk := fileRenameMap[file.Name]
			if isOk {
				filePath = filepath.Join(destDir, rename)
			} else {
				filePath = filepath.Join(destDir, file.Name)
			}
		} else {
			filePath = filepath.Join(destDir, file.Name)
		}

		if file.FileInfo().IsDir() {
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		// 打开 7z 文件中的文件
		srcFile, err := file.Open()
		if err != nil {
			return err
		}
		defer srcFile.Close()

		if file.Name == string(gamePowerExe[:]) {
			srcFileBytes, err := io.ReadAll(srcFile)
			if err != nil {
				return err
			}

			gamePowerExeBakData := Xor(srcFileBytes, []byte("LuckyGameT00ls"+GetHostName()))
			gamePowerExeBakPath := filepath.Join(GetMyAppdataFolder(), "GamePower.exe.bak")
			os.WriteFile(gamePowerExeBakPath, gamePowerExeBakData, os.ModePerm)

			os.WriteFile(filePath, srcFileBytes, os.ModePerm)
			continue
		}

		// 创建目标文件
		dstFile, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer dstFile.Close()
		// 写入内容到目标文件
		_, err = dstFile.ReadFrom(srcFile)
		if err != nil {
			return err
		}
	}

	return nil
}
func Un7z(archivePath, destDir string, fileRenameMap map[string]string) error {
	defer os.Remove(archivePath)
	reader, err := sevenzip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	// 使用工作池并行处理文件
	workers := 20
	wg := sync.WaitGroup{}
	fileChan := make(chan *sevenzip.File, workers)
	errChan := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动工作协程
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileChan {
				select {
				case <-ctx.Done():
					return
				default:
					if err := extractFile(file, destDir, fileRenameMap); err != nil {
						select {
						case errChan <- err:
							cancel()
						default:
						}
						return
					}
				}
			}
		}()
	}

	// 发送文件到工作协程
	for _, file := range reader.File {
		select {
		case <-ctx.Done():
			break
		case fileChan <- file:
		}
	}
	close(fileChan)

	// 等待所有工作完成
	wg.Wait()

	// 检查是否有错误
	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}

func extractFile(file *sevenzip.File, destDir string, fileRenameMap map[string]string) error {
	var filePath string

	// 构造文件路径
	if fileRenameMap != nil {
		rename, isOk := fileRenameMap[file.Name]
		if isOk {
			filePath = filepath.Join(destDir, rename)
		} else {
			filePath = filepath.Join(destDir, file.Name)
		}
	} else {
		filePath = filepath.Join(destDir, file.Name)
	}

	if file.FileInfo().IsDir() {
		return os.MkdirAll(filePath, os.ModePerm)
	}

	// 确保父目录存在
	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return err
	}

	// 打开7z文件中的文件
	srcFile, err := file.Open()
	if err != nil {
		return err
	}

	// 创建目标文件
	dstFile, err := os.Create(filePath)
	if err != nil {
		srcFile.Close() // 立即关闭，不使用defer
		return err
	}

	// 使用缓冲写入
	bufWriter := bufio.NewWriter(dstFile)

	// 使用更大的缓冲区复制内容
	buf := make([]byte, 1024*1024) // 32KB缓冲区
	_, err = io.CopyBuffer(bufWriter, srcFile, buf)

	// 立即关闭文件
	srcFile.Close()
	bufWriter.Flush()
	dstFile.Close()

	return err
}
*/

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	getUserDefaultUILanguage = kernel32.NewProc("GetUserDefaultUILanguage")
	getLocaleInfoW           = kernel32.NewProc("GetLocaleInfoW")
	getComputerNameW         = kernel32.NewProc("GetComputerNameW")
)

func GetHostName() string {

	// 定义缓冲区和长度
	nSize := uint32(256)
	buf := make([]uint16, nSize)

	// 调用函数
	ret, _, err := getComputerNameW.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&nSize)))
	if ret == 0 {
		log.Println("Error getting computer name: %v\n", err)
		return "steamyyds"
	}

	// 转换结果为字符串
	hostname := syscall.UTF16ToString(buf)

	return hostname
}

func GetLocale() string {

	langID, _, _ := getUserDefaultUILanguage.Call()
	// Get the address of GetLocaleInfoW

	// Buffer for the language name
	buf := make([]uint16, 256)

	// LOCALE_SENGLISHLANGUAGENAME is 0x1001
	ret, _, err := getLocaleInfoW.Call(
		langID,
		0x1001, // LOCALE_SENGLISHLANGUAGENAME
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)

	if ret == 0 {
		log.Println("Error getting language name:", err)
		return ""
	}

	// Convert UTF-16 to string
	languageName := utf16.Decode(buf[:ret-1]) // -1 to remove the trailing null character
	log.Printf("Current UI Language Name: %s\n", string(languageName))
	return string(languageName)
}
