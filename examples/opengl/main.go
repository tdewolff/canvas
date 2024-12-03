//go:build cgo

package main

import (
	"fmt"
	"runtime"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/Seanld/canvas"
	"github.com/Seanld/canvas/renderers/opengl"
)

func main() {
	runtime.LockOSThread()

	opengl := opengl.New(200.0, 100.0, canvas.DPMM(5.0))
	ctx := canvas.NewContext(opengl)
	if err := canvas.DrawPreview(ctx); err != nil {
		panic(err)
	}

	// Set up window
	if err := glfw.Init(); err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	width, height := 800, 400
	window, err := glfw.CreateWindow(width, height, "tdewolff/canvas OpenGL demo", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()
	window.SetKeyCallback(onKey)

	if err := gl.Init(); err != nil {
		panic(err)
	}
	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Println("OpenGL version", version)

	// Compile canvas for OpenGl
	opengl.Compile()

	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	gl.ClearColor(1, 1, 1, 1)
	for !window.ShouldClose() {
		gl.Clear(gl.COLOR_BUFFER_BIT)

		// Draw compiled canvas to OpenGL
		opengl.Draw()

		glfw.PollEvents()
		window.SwapBuffers()
	}
}

func onKey(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if action == glfw.Press && (key == glfw.KeyEscape || key == glfw.KeyQ) {
		w.SetShouldClose(true)
	}
}
