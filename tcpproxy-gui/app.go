package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/energye/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var g_Tasks = []TaskConf{
	{Name: "fedora", Host: "192.168.100.128", Ports: []Port{
		{From: 5510, To: 5510},
		{From: 5710, To: 5710},
		{From: 5910, To: 5910},
	}},
	{Name: "ubuntu", Host: "192.168.100.34", Ports: []Port{
		{From: 5510, To: 5510},
		{From: 5710, To: 5710},
		{From: 5910, To: 5910},
	}},
}

// App struct
type App struct {
	ctx     context.Context
	tasks   map[string]*ProxyTask
	w       sync.WaitGroup
	canQuit bool
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		tasks: map[string]*ProxyTask{},
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	systray.Run(a.systemTray, func() {})
}

// Greet returns a greeting for the given name
func (a *App) StartTask(name string, host string, ports []Port) error {
	if a.IsTaskRun(name) {
		return fmt.Errorf("Task %s already running", name)
	}
	task := NewProxyTask(name, host, ports)
	a.tasks[name] = task
	err := task.Start(&a.w, func() {
		delete(a.tasks, name)
	})
	if err != nil {
		fmt.Printf("Task %s start", name)
	}
	return err
}
func (a *App) StopTask(name string) {
	if t := a.tasks[name]; t != nil {
		t.Stop()
	}
}
func (a *App) IsTaskRun(name string) bool {
	_, ok := a.tasks[name]
	return ok
}
func (a *App) GetAllTask() []TaskConf {
	return g_Tasks
}
func (a *App) SaveTask(task TaskConf) {
	defer saveConfig()
	for i, t := range g_Tasks {
		if t.Name == task.Name {
			g_Tasks[i] = task
			return
		}
	}
	g_Tasks = append(g_Tasks, task)
}
func (a *App) DelTask(name string) {
	a.StopTask(name)
	for i, t := range g_Tasks {
		if t.Name == name {
			g_Tasks = append(g_Tasks[:i], g_Tasks[i+1:]...)
			return
		}
	}
}

func (a *App) GetTask(name string) *TaskConf {
	for _, t := range g_Tasks {
		if t.Name == name {
			return &t
		}
	}
	return nil
}

//go:embed build/windows/icon.ico
var icon []byte

func (a *App) systemTray() {
	systray.SetIcon(icon) // read the icon from a file
	systray.AddMenuItem("Hide", "Hide The Window").Click(func() { runtime.WindowHide(a.ctx) })
	show := systray.AddMenuItem("Show", "Show The Window")
	systray.AddSeparator()
	exit := systray.AddMenuItem("Exit", "Quit The Program")

	show.Click(func() { runtime.WindowShow(a.ctx) })
	exit.Click(func() { os.Exit(0) })

	systray.SetOnDClick(func(menu systray.IMenu) { runtime.WindowShow(a.ctx) })
	systray.SetOnRClick(func(menu systray.IMenu) { menu.ShowMenu() })
	systray.SetTooltip("tcp协议代理工具")
	systray.SetTitle("tcp协议代理工具")
}

func (a *App) beforeClose(ctx context.Context) bool {
	if a.canQuit {
		return false
	}
	runtime.EventsEmit(ctx, "queryQuit")
	return true
}
func (a *App) Quit() {
	a.canQuit = true
	runtime.Quit(a.ctx)
}

func (a *App) onShutdown(ctx context.Context) {
	saveConfig()
}

func saveConfig() {
	buff, err := json.Marshal(g_Tasks)
	if err != nil {
		return
	}
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	os.WriteFile(filepath.Join(dir, "config.json"), buff, 0777)
}
func init() {
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	buff, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		return
	}
	json.Unmarshal(buff, &g_Tasks)
}
