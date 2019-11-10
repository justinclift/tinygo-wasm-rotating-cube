package main

import (
	"syscall/js"
	"unsafe"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/justinclift/webgl"
)

var (
	gl          js.Value
	movMatrix   mgl32.Mat4
	ModelMatrix js.Value
	tmark       float64
	rotation    float32
)

// * BUFFERS + SHADERS *
// Shamelessly copied from https://www.tutorialspoint.com/webgl/webgl_cube_rotation.htm //
var verticesNative = []float32{
	-1, -1, -1, 1, -1, -1, 1, 1, -1, -1, 1, -1,
	-1, -1, 1, 1, -1, 1, 1, 1, 1, -1, 1, 1,
	-1, -1, -1, -1, 1, -1, -1, 1, 1, -1, -1, 1,
	1, -1, -1, 1, 1, -1, 1, 1, 1, 1, -1, 1,
	-1, -1, -1, -1, -1, 1, 1, -1, 1, 1, -1, -1,
	-1, 1, -1, -1, 1, 1, 1, 1, 1, 1, 1, -1,
}
var colorsNative = []float32{
	5, 3, 7, 5, 3, 7, 5, 3, 7, 5, 3, 7,
	1, 1, 3, 1, 1, 3, 1, 1, 3, 1, 1, 3,
	0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1,
	1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0,
	1, 1, 0, 1, 1, 0, 1, 1, 0, 1, 1, 0,
	0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0,
}
var indicesNative = []uint16{
	0, 1, 2, 0, 2, 3, 4, 5, 6, 4, 6, 7,
	8, 9, 10, 8, 10, 11, 12, 13, 14, 12, 14, 15,
	16, 17, 18, 16, 18, 19, 20, 21, 22, 20, 22, 23,
}

const vertShaderCode = `
attribute vec3 position;
uniform mat4 Pmatrix;
uniform mat4 Vmatrix;
uniform mat4 Mmatrix;
attribute vec3 color;
varying vec3 vColor;

void main(void) {
	gl_Position = Pmatrix*Vmatrix*Mmatrix*vec4(position, 1.);
	vColor = color;
}
`
const fragShaderCode = `
precision mediump float;
varying vec3 vColor;
void main(void) {
	gl_FragColor = vec4(vColor, 1.);
}
`

func main() {
	// Init Canvas
	doc := js.Global().Get("document")
	canvasEl := doc.Call("getElementById", "gocanvas")
	width := canvasEl.Get("clientWidth").Int()
	height := canvasEl.Get("clientHeight").Int()
	canvasEl.Call("setAttribute", "width", width)
	canvasEl.Call("setAttribute", "height", height)
	canvasEl.Set("tabIndex", 0) // Not sure if this is needed

	gl = canvasEl.Call("getContext", "webgl")
	if gl == js.Undefined() {
		gl = canvasEl.Call("getContext", "experimental-webgl")
	}
	// once again
	if gl == js.Undefined() {
		js.Global().Call("alert", "browser might not support webgl")
		return
	}

	// Convert buffers to JS TypedArrays
	var colors = webgl.SliceToTypedArray(colorsNative)
	var vertices = webgl.SliceToTypedArray(verticesNative)
	var indices = webgl.SliceToTypedArray(indicesNative)

	// Create vertex buffer
	vertexBuffer := gl.Call("createBuffer")
	gl.Call("bindBuffer", webgl.ARRAY_BUFFER, vertexBuffer)
	gl.Call("bufferData", webgl.ARRAY_BUFFER, vertices, webgl.STATIC_DRAW)

	// Create color buffer
	colorBuffer := gl.Call("createBuffer")
	gl.Call("bindBuffer", webgl.ARRAY_BUFFER, colorBuffer)
	gl.Call("bufferData", webgl.ARRAY_BUFFER, colors, webgl.STATIC_DRAW)

	// Create index buffer
	indexBuffer := gl.Call("createBuffer")
	gl.Call("bindBuffer", webgl.ELEMENT_ARRAY_BUFFER, indexBuffer)
	gl.Call("bufferData", webgl.ELEMENT_ARRAY_BUFFER, indices, webgl.STATIC_DRAW)

	// * Shaders *

	// Create a vertex shader object
	vertShader := gl.Call("createShader", webgl.VERTEX_SHADER)
	gl.Call("shaderSource", vertShader, vertShaderCode)
	gl.Call("compileShader", vertShader)

	// Create fragment shader object
	fragShader := gl.Call("createShader", webgl.FRAGMENT_SHADER)
	gl.Call("shaderSource", fragShader, fragShaderCode)
	gl.Call("compileShader", fragShader)

	// Create a shader program object to store the combined shader program
	shaderProgram := gl.Call("createProgram")
	gl.Call("attachShader", shaderProgram, vertShader)
	gl.Call("attachShader", shaderProgram, fragShader)
	gl.Call("linkProgram", shaderProgram)

	// Associate attributes to vertex shader
	PositionMatrix := gl.Call("getUniformLocation", shaderProgram, "Pmatrix")
	ViewMatrix := gl.Call("getUniformLocation", shaderProgram, "Vmatrix")
	ModelMatrix = gl.Call("getUniformLocation", shaderProgram, "Mmatrix")

	gl.Call("bindBuffer", webgl.ARRAY_BUFFER, vertexBuffer)
	position := gl.Call("getAttribLocation", shaderProgram, "position")
	gl.Call("vertexAttribPointer", position, 3, webgl.FLOAT, false, 0, 0)
	gl.Call("enableVertexAttribArray", position)

	gl.Call("bindBuffer", webgl.ARRAY_BUFFER, colorBuffer)
	color := gl.Call("getAttribLocation", shaderProgram, "color")
	gl.Call("vertexAttribPointer", color, 3, webgl.FLOAT, false, 0, 0)
	gl.Call("enableVertexAttribArray", color)

	gl.Call("useProgram", shaderProgram)

	// Set WebGL properties
	gl.Call("clearColor", 0.5, 0.5, 0.5, 0.9) // Color the screen is cleared to
	gl.Call("clearDepth", 1.0)                // Z value that is set to the Depth buffer every frame
	gl.Call("viewport", 0, 0, width, height)  // Viewport size
	gl.Call("depthFunc", webgl.LEQUAL)

	// * Create Matrixes *
	ratio := float32(width) / float32(height)

	// Generate and apply projection matrix
	projMatrix := mgl32.Perspective(mgl32.DegToRad(45.0), ratio, 1, 100.0)
	var projMatrixBuffer *[16]float32
	projMatrixBuffer = (*[16]float32)(unsafe.Pointer(&projMatrix))
	typedProjMatrixBuffer := webgl.SliceToTypedArray([]float32((*projMatrixBuffer)[:]))
	gl.Call("uniformMatrix4fv", PositionMatrix, false, typedProjMatrixBuffer)

	// Generate and apply view matrix
	viewMatrix := mgl32.LookAtV(mgl32.Vec3{3.0, 3.0, 3.0}, mgl32.Vec3{0.0, 0.0, 0.0}, mgl32.Vec3{0.0, 1.0, 0.0})
	var viewMatrixBuffer *[16]float32
	viewMatrixBuffer = (*[16]float32)(unsafe.Pointer(&viewMatrix))
	typedViewMatrixBuffer := webgl.SliceToTypedArray([]float32((*viewMatrixBuffer)[:]))
	gl.Call("uniformMatrix4fv", ViewMatrix, false, typedViewMatrixBuffer)

	// * Drawing the Cube *
	movMatrix = mgl32.Ident4()

	// Bind to element array for draw function
	gl.Call("bindBuffer", webgl.ELEMENT_ARRAY_BUFFER, indexBuffer)

	// Start the frame renderer
	js.Global().Call("requestAnimationFrame", js.Global().Get("renderFrame"))
}

// Renders one frame of the animation
//go:export renderFrame
func renderFrame(now float64) {
	// Calculate rotation rate
	tdiff := now - tmark
	tmark = now
	rotation = rotation + float32(tdiff)/500

	// Do new model matrix calculations
	movMatrix = mgl32.HomogRotate3DX(0.5 * rotation)
	movMatrix = movMatrix.Mul4(mgl32.HomogRotate3DY(0.3 * rotation))
	movMatrix = movMatrix.Mul4(mgl32.HomogRotate3DZ(0.2 * rotation))

	// Convert model matrix to a JS TypedArray
	var modelMatrixBuffer *[16]float32
	modelMatrixBuffer = (*[16]float32)(unsafe.Pointer(&movMatrix))
	typedModelMatrixBuffer := webgl.SliceToTypedArray([]float32((*modelMatrixBuffer)[:]))

	// Apply the model matrix
	gl.Call("uniformMatrix4fv", ModelMatrix, false, typedModelMatrixBuffer)

	// Clear the screen
	gl.Call("enable", webgl.DEPTH_TEST)
	gl.Call("clear", webgl.COLOR_BUFFER_BIT)
	gl.Call("clear", webgl.DEPTH_BUFFER_BIT)

	// Draw the cube
	gl.Call("drawElements", webgl.TRIANGLES, len(indicesNative), webgl.UNSIGNED_SHORT, 0)

	// Keep the frame rendering going
	js.Global().Call("requestAnimationFrame", js.Global().Get("renderFrame"))
}
