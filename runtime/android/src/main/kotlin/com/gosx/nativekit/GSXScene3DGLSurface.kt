package com.gosx.nativekit

import android.content.Context
import android.opengl.GLES20
import android.opengl.GLSurfaceView
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.viewinterop.AndroidView
import java.nio.ByteBuffer
import java.nio.ByteOrder
import javax.microedition.khronos.egl.EGLConfig
import javax.microedition.khronos.opengles.GL10
import kotlin.math.cos
import kotlin.math.max
import kotlin.math.min
import kotlin.math.sin

@Composable
internal fun GSXScene3DNativeSurface(scene: GSXScene3DScene, modifier: Modifier = Modifier) {
    AndroidView(
        modifier = modifier,
        factory = { context ->
            GSXScene3DGLSurfaceView(context).also { it.setScene(scene) }
        },
        update = { view ->
            view.setScene(scene)
        },
    )
}

private class GSXScene3DGLSurfaceView(context: Context) : GLSurfaceView(context) {
    private val sceneRenderer = GSXScene3DGLRenderer()

    init {
        setEGLContextClientVersion(2)
        setRenderer(sceneRenderer)
        renderMode = RENDERMODE_WHEN_DIRTY
    }

    fun setScene(scene: GSXScene3DScene) {
        queueEvent {
            sceneRenderer.scene = scene
        }
        requestRender()
    }
}

private class GSXScene3DGLRenderer : GLSurfaceView.Renderer {
    @Volatile
    var scene: GSXScene3DScene = GSXScene3DScene()

    private var program = 0
    private var positionHandle = 0
    private var colorHandle = 0
    private var pointSizeHandle = 0

    override fun onSurfaceCreated(gl: GL10?, config: EGLConfig?) {
        program = createProgram(vertexShaderSource, fragmentShaderSource)
        positionHandle = GLES20.glGetAttribLocation(program, "aPosition")
        colorHandle = GLES20.glGetUniformLocation(program, "uColor")
        pointSizeHandle = GLES20.glGetUniformLocation(program, "uPointSize")
        GLES20.glEnable(GLES20.GL_BLEND)
        GLES20.glBlendFunc(GLES20.GL_SRC_ALPHA, GLES20.GL_ONE_MINUS_SRC_ALPHA)
    }

    override fun onSurfaceChanged(gl: GL10?, width: Int, height: Int) {
        GLES20.glViewport(0, 0, width, height)
    }

    override fun onDrawFrame(gl: GL10?) {
        val current = scene
        val background = glColor(current.background, alpha = 1f)
        GLES20.glClearColor(background[0], background[1], background[2], background[3])
        GLES20.glClear(GLES20.GL_COLOR_BUFFER_BIT or GLES20.GL_DEPTH_BUFFER_BIT)
        GLES20.glUseProgram(program)

        val renderableNodes = current.nodes.filter {
            it.tag == "mesh" || it.tag == "model" || it.tag == "points" || it.tag == "instancedMesh" || it.tag == "computeParticles"
        }
        renderableNodes.forEachIndexed { index, node ->
            drawNode(node, index, renderableNodes.size)
        }
        drawPostEffects(current.postEffects)
    }

    private fun drawNode(node: GSXScene3DNode, index: Int, total: Int) {
        when (node.tag) {
            "instancedMesh" -> drawInstanced(node, index, total)
            "computeParticles" -> drawComputeParticles(node, index, total)
            "points" -> drawPoints(node, index, total)
            "model" -> drawRect(nodeRect(node, index, total), glColor(node.color, alpha = 0.82f))
            else -> drawRect(nodeRect(node, index, total), glColor(node.color, alpha = 0.88f))
        }
    }

    private fun drawInstanced(node: GSXScene3DNode, index: Int, total: Int) {
        val parent = nodeRect(node, index, total)
        val count = max(node.count, 1)
        val width = max(parent.width * 0.42f, 0.05f)
        val height = max(parent.height * 0.42f, 0.05f)
        repeat(count) { i ->
            val offset = (i.toFloat() - (count - 1).toFloat() / 2f) * width * 0.72f
            val rise = ((i % 2) * 2 - 1).toFloat() * height * 0.18f
            drawRect(
                GLRect(
                    centerX = parent.centerX + offset,
                    centerY = parent.centerY + rise,
                    width = width,
                    height = height,
                ),
                glColor(node.color, alpha = 0.84f),
            )
        }
    }

    private fun drawPoints(node: GSXScene3DNode, index: Int, total: Int) {
        val parent = nodeRect(node, index, total)
        val count = max(node.count, 1)
        val radius = max(node.size.toFloat() * 0.025f, 0.018f)
        repeat(count) { i ->
            val offset = (i - count / 2).toFloat() * radius * 2.4f
            drawPoint(parent.centerX + offset, parent.centerY, radius * 420f, glColor(node.color, alpha = 0.9f))
        }
    }

    private fun drawComputeParticles(node: GSXScene3DNode, index: Int, total: Int) {
        val parent = nodeRect(node, index, total)
        val count = max(min(node.count, 48), 1)
        val radius = max(node.size.toFloat() * 0.025f, 0.014f)
        repeat(count) { i ->
            val angle = i.toFloat() * 0.62f
            val spiral = i.toFloat() / count.toFloat() * max(parent.width, parent.height) * 0.5f
            drawPoint(
                x = parent.centerX + cos(angle) * spiral,
                y = parent.centerY + sin(angle) * spiral * 0.6f,
                pointSize = radius * 420f,
                color = glColor(node.color, alpha = 0.72f),
            )
        }
    }

    private fun drawPostEffects(effects: List<GSXScene3DPostEffect>) {
        effects.forEach { effect ->
            when (effect.kind) {
                "bloom" -> {
                    val alpha = (effect.intensity * 0.16).coerceIn(0.0, 0.34).toFloat()
                    if (alpha > 0f) {
                        drawRect(GLRect(0f, 0.92f, 2f, 0.08f), floatArrayOf(1f, 1f, 1f, alpha))
                        drawRect(GLRect(0f, -0.92f, 2f, 0.08f), floatArrayOf(1f, 1f, 1f, alpha))
                    }
                }
                "vignette" -> {
                    val alpha = (effect.intensity * 0.46).coerceIn(0.0, 0.58).toFloat()
                    if (alpha > 0f) {
                        drawRect(GLRect(0f, 0.9f, 2f, 0.2f), floatArrayOf(0f, 0f, 0f, alpha))
                        drawRect(GLRect(0f, -0.9f, 2f, 0.2f), floatArrayOf(0f, 0f, 0f, alpha))
                    }
                }
                "colorGrade" -> {
                    val alpha = (kotlin.math.abs(effect.saturation - 1.0) * 0.08 +
                        kotlin.math.abs(effect.contrast - 1.0) * 0.05 +
                        kotlin.math.abs(effect.exposure) * 0.04).coerceIn(0.0, 0.18).toFloat()
                    if (alpha > 0f) {
                        val tint = if (effect.saturation >= 1.0) {
                            floatArrayOf(1f, 0.82f, 0.54f, alpha)
                        } else {
                            floatArrayOf(0.46f, 0.72f, 1f, alpha)
                        }
                        drawRect(GLRect(0f, 0f, 2f, 2f), tint)
                    }
                }
                "toneMapping" -> {
                    val alpha = (kotlin.math.abs(effect.exposure - 1.0) * 0.08).coerceIn(0.0, 0.2).toFloat()
                    if (alpha > 0f) {
                        val value = if (effect.exposure >= 1.0) 1f else 0f
                        drawRect(GLRect(0f, 0f, 2f, 2f), floatArrayOf(value, value, value, alpha))
                    }
                }
            }
        }
    }

    private fun nodeRect(node: GSXScene3DNode, index: Int, total: Int): GLRect {
        val slotCount = max(total, 1)
        val centerX = -1f + (2f * (index + 1).toFloat() / (slotCount + 1).toFloat()) + node.x.toFloat() * 0.08f
        val centerY = node.y.toFloat() * 0.12f + node.z.toFloat() * 0.02f
        return GLRect(
            centerX = centerX,
            centerY = centerY,
            width = max(node.width.toFloat() * 0.18f, 0.08f),
            height = max(node.height.toFloat() * 0.18f, 0.08f),
        )
    }

    private fun drawRect(rect: GLRect, color: FloatArray) {
        val left = rect.centerX - rect.width / 2f
        val right = rect.centerX + rect.width / 2f
        val bottom = rect.centerY - rect.height / 2f
        val top = rect.centerY + rect.height / 2f
        drawVertices(
            floatArrayOf(
                left, bottom,
                right, bottom,
                left, top,
                right, top,
            ),
            GLES20.GL_TRIANGLE_STRIP,
            color,
            pointSize = 1f,
        )
    }

    private fun drawPoint(x: Float, y: Float, pointSize: Float, color: FloatArray) {
        drawVertices(floatArrayOf(x, y), GLES20.GL_POINTS, color, pointSize)
    }

    private fun drawVertices(vertices: FloatArray, mode: Int, color: FloatArray, pointSize: Float) {
        val buffer = ByteBuffer.allocateDirect(vertices.size * 4)
            .order(ByteOrder.nativeOrder())
            .asFloatBuffer()
            .apply {
                put(vertices)
                position(0)
            }
        GLES20.glUniform4fv(colorHandle, 1, color, 0)
        GLES20.glUniform1f(pointSizeHandle, pointSize)
        GLES20.glEnableVertexAttribArray(positionHandle)
        GLES20.glVertexAttribPointer(positionHandle, 2, GLES20.GL_FLOAT, false, 0, buffer)
        GLES20.glDrawArrays(mode, 0, vertices.size / 2)
        GLES20.glDisableVertexAttribArray(positionHandle)
    }
}

private data class GLRect(
    val centerX: Float,
    val centerY: Float,
    val width: Float,
    val height: Float,
)

private fun glColor(hex: String, alpha: Float): FloatArray {
    val cleaned = hex.trim().removePrefix("#")
    val value = cleaned.toLongOrNull(16) ?: return floatArrayOf(16f / 255f, 24f / 255f, 32f / 255f, alpha)
    return when (cleaned.length) {
        6 -> floatArrayOf(
            ((value shr 16) and 0xffL).toFloat() / 255f,
            ((value shr 8) and 0xffL).toFloat() / 255f,
            (value and 0xffL).toFloat() / 255f,
            alpha,
        )
        8 -> floatArrayOf(
            ((value shr 16) and 0xffL).toFloat() / 255f,
            ((value shr 8) and 0xffL).toFloat() / 255f,
            (value and 0xffL).toFloat() / 255f,
            (((value shr 24) and 0xffL).toFloat() / 255f) * alpha,
        )
        else -> floatArrayOf(16f / 255f, 24f / 255f, 32f / 255f, alpha)
    }
}

private fun createProgram(vertexSource: String, fragmentSource: String): Int {
    val vertexShader = loadShader(GLES20.GL_VERTEX_SHADER, vertexSource)
    val fragmentShader = loadShader(GLES20.GL_FRAGMENT_SHADER, fragmentSource)
    return GLES20.glCreateProgram().also { program ->
        GLES20.glAttachShader(program, vertexShader)
        GLES20.glAttachShader(program, fragmentShader)
        GLES20.glLinkProgram(program)
    }
}

private fun loadShader(type: Int, source: String): Int =
    GLES20.glCreateShader(type).also { shader ->
        GLES20.glShaderSource(shader, source)
        GLES20.glCompileShader(shader)
    }

private const val vertexShaderSource = """
attribute vec2 aPosition;
uniform float uPointSize;

void main() {
    gl_Position = vec4(aPosition, 0.0, 1.0);
    gl_PointSize = uPointSize;
}
"""

private const val fragmentShaderSource = """
precision mediump float;
uniform vec4 uColor;

void main() {
    gl_FragColor = uColor;
}
"""
