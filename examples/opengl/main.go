// +build cgo

package main

import (
	"log"
	"runtime"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/tdewolff/canvas"
)

var vertexShaderSource = `
    #version 410
    in vec3 position;
	in vec2 texcoord;

	out vec2 Texcoord;

    void main() {
        gl_Position = vec4(position, 1.0);
		Texcoord = texcoord;
    }
` + "\x00"

var fragmentShaderSource = `
    #version 410
	in vec2 Texcoord;

    out vec4 frag_colour;

    void main() {
		vec2 p = Texcoord.st;
		vec2 px = dFdx(p);
		vec2 py = dFdy(p);
		float fx = (2*p.x)*px.x - px.y;
		float fy = (2*p.x)*py.x - py.y;
		float sd = (p.x*p.x - p.y)/sqrt(fx*fx + fy*fy);
		float alpha = 0.5 - sd;
		if (alpha >= 1)
			frag_colour = vec4(1, 1, 1, 1);
		else if (alpha <= 0)
			discard;
		else
			frag_colour = vec4(1, 1, 1, alpha);
    }
` + "\x00"

func main() {
	runtime.LockOSThread()

	fontFamily := canvas.NewFontFamily("dejavu-serif")
	fontFamily.Use(canvas.CommonLigatures)
	if err := fontFamily.LoadLocalFont("DejaVuSerif", canvas.FontRegular); err != nil {
		panic(err)
	}

	//p, _ := canvas.ParseSVG("M0 0.5L0.5 0.5Q1 0.5 1 0Q1 -0.5 0.5 -0.5L-0.5 -0.5C-1 -0.5 -1 0.5 -0.5 0.5z")
	p, _ := canvas.ParseSVG("M0 0.5L0.5 0.5C1 0.5 1 -0.5 0.5 -0.5L-0.5 -0.5C-1 -0.5 -1 0.5 -0.5 0.5z")
	c := canvas.New(0.0, 0.0)
	c.SetFillColor(canvas.Blue)
	c.DrawPath(0.0, 0.0, p)
	ogl := c.ToOpenGL()

	if err := glfw.Init(); err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	//glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4) // OR 2
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	width, height := 800, 800
	window, err := glfw.CreateWindow(width, height, "Canvas OpenGL demo", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()
	window.SetKeyCallback(onKey)

	if err := gl.Init(); err != nil {
		panic(err)
	}
	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Println("OpenGL version", version)

	ogl.Compile()

	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	gl.ClearColor(1, 1, 1, 1)
	for !window.ShouldClose() {
		gl.Clear(gl.COLOR_BUFFER_BIT)

		ogl.Draw()

		glfw.PollEvents()
		window.SwapBuffers()
	}
}

func onKey(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if action == glfw.Press && (key == glfw.KeyEscape || key == glfw.KeyQ) {
		w.SetShouldClose(true)
	}
}
