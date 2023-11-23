//go:generate goversioninfo
package main

import (
	_ "embed"
	"log/slog"
	"os"
	"syscall"
	"unsafe"

	"github.com/energye/systray"
	"golang.org/x/sys/windows"
)

const (
	ES_SYSTEM_REQUIRED   = 0x00000001
	ES_DISPLAY_REQUIRED  = 0x00000002
	ES_USER_PRESENT      = 0x00000004
	ES_AWAYMODE_REQUIRED = 0x00000040
	ES_CONTINUOUS        = 0x80000000

	SPI_SETSCREENSAVEACTIVE = 0x0011
	SPIF_SENDCHANGE         = 2
	SPIF_SENDWININICHANGE   = SPIF_SENDCHANGE
)

var (
	setThreadExecutionState *windows.LazyProc
	systemParametersInfo    *windows.LazyProc
	createMutex             *windows.LazyProc
	releaseMutex            *windows.LazyProc
	openMutex               *windows.LazyProc
)

//go:embed nosleep.ico
var icon []byte

type SECURITY_ATTRIBUTES struct {
	Length             uint32 //DWORD
	SecurityDescriptor *byte  //LPVOID
	InheritHandle      int32  //BOOL
}

func init() {
	libkernel32 := windows.NewLazySystemDLL("kernel32.dll")
	setThreadExecutionState = libkernel32.NewProc("SetThreadExecutionState")
	createMutex = libkernel32.NewProc("CreateMutexW")
	releaseMutex = libkernel32.NewProc("ReleaseMutex")
	openMutex = libkernel32.NewProc("OpenMutexW")

	libuser32 := windows.NewLazySystemDLL("user32.dll")
	systemParametersInfo = libuser32.NewProc("SystemParametersInfoW")

}
func SetThreadExecutionState(hWnd windows.Handle) windows.Handle {
	ret, _, _ := syscall.SyscallN(setThreadExecutionState.Addr(), uintptr(hWnd))

	return windows.Handle(ret)
}

func SystemParametersInfo(uiAction, uiParam uintptr, PVOID, fWinIni uintptr) windows.Handle {
	ret, _, _ := syscall.SyscallN(systemParametersInfo.Addr(), uintptr(uiAction), uintptr(uiParam), uintptr(PVOID), uintptr(fWinIni))
	return windows.Handle(ret)
}

func CreateMutex(lpMutexAttributes *SECURITY_ATTRIBUTES, bInitialOwner bool, lpName string) windows.Handle {
	_p1 := uint32(0)
	if bInitialOwner {
		_p1 = 1
	}
	_p2, err := syscall.UTF16PtrFromString(lpName)
	if err != nil {
		return 0
	}
	ret, _, _ := syscall.SyscallN(createMutex.Addr(), uintptr(unsafe.Pointer(lpMutexAttributes)), uintptr(_p1), uintptr(unsafe.Pointer(_p2)))
	return windows.Handle(ret)
}
func ReleaseMutex(hMutex windows.Handle) bool {
	ret, _, _ := syscall.SyscallN(releaseMutex.Addr(), uintptr(hMutex))
	return ret != 0
}

func OpenMutex(dwDesiredAccess uint32, bInheritHandle bool, lpName string) windows.Handle {
	_p1 := uint32(0)
	if bInheritHandle {
		_p1 = 1
	}
	_p2, err := syscall.UTF16PtrFromString(lpName)
	if err != nil {
		return 0
	}
	ret, _, _ := syscall.SyscallN(openMutex.Addr(), uintptr(dwDesiredAccess), uintptr(_p1), uintptr(unsafe.Pointer(_p2)))
	return windows.Handle(ret)
}

func BlockSleep(noSleep bool) {
	slog.Info("noSleep", "value", noSleep)
	if noSleep {
		// DisableSCR
		SystemParametersInfo(SPI_SETSCREENSAVEACTIVE, 0, 0, SPIF_SENDWININICHANGE)
		// try this for vista, it will fail on XP
		if SetThreadExecutionState(ES_CONTINUOUS|ES_SYSTEM_REQUIRED|ES_DISPLAY_REQUIRED) == 0 {
			SetThreadExecutionState(ES_CONTINUOUS | ES_SYSTEM_REQUIRED)
		}
	} else {
		// EnableSCR
		SystemParametersInfo(SPI_SETSCREENSAVEACTIVE, 1, 0, SPIF_SENDWININICHANGE)
		SetThreadExecutionState(ES_CONTINUOUS)
	}
}

func main() {
	// 只允许一个实例运行
	hMutex := CreateMutex(nil, true, "nosleep")
	if hMutex == 0 {
		text, _ := syscall.UTF16PtrFromString("CreateMutex failed")

		windows.MessageBox(0, text, nil, windows.MB_ICONERROR)
	}
	event, err := windows.WaitForSingleObject(hMutex, 0)
	if err != nil {
		slog.Error("WaitForSingleObject", "error", err)
		os.Exit(1)
	}
	if event == windows.WAIT_OBJECT_0 || event == windows.WAIT_ABANDONED {
		defer ReleaseMutex(hMutex)
		systray.Run(onReady, onExit)
	} else {
		text, _ := syscall.UTF16PtrFromString("nosleep is already running")
		windows.MessageBox(0, text, nil, windows.MB_ICONERROR)
	}
}
func onReady() {
	systray.SetIcon(icon)
	m := systray.AddMenuItemCheckbox("阻止锁屏", "阻止电脑睡眠", true)
	m.Click(func() {
		if m.Checked() {
			m.Uncheck()
		} else {
			m.Check()
		}
		BlockSleep(m.Checked())
	})
	systray.AddMenuItem("退出", "退出程序").Click(func() {
		systray.Quit()
	})
	systray.SetTooltip("阻止电脑睡眠工具")
	systray.SetTitle("NoSleep")

	BlockSleep(true)
}
func onExit() {
	BlockSleep(false)
	slog.Info("Exit")
}
