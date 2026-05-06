package com.gosx.nativekit

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.text.BasicText
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlin.math.cos
import kotlin.math.max
import kotlin.math.roundToInt
import kotlin.math.sin

data class GSXScene3DScene(
    val width: Double = 640.0,
    val height: Double = 360.0,
    val background: String = "#101820",
    val postEffects: List<GSXScene3DPostEffect> = emptyList(),
    val htmlOverlays: List<GSXScene3DHTMLOverlay> = emptyList(),
    val nodes: List<GSXScene3DNode> = emptyList(),
)

data class GSXScene3DHTMLOverlay(
    val id: String,
    val html: String,
    val className: String = "",
    val x: Double = 0.0,
    val y: Double = 0.0,
    val z: Double = 0.0,
    val width: Double = 1.8,
    val height: Double = 0.72,
    val opacity: Double = 1.0,
    val offsetX: Double = 0.0,
    val offsetY: Double = 0.0,
    val pointerEvents: String = "none",
)

data class GSXScene3DPostEffect(
    val kind: String,
    val threshold: Double = 0.0,
    val intensity: Double = 0.0,
    val radius: Double = 0.0,
    val scale: Double = 0.0,
    val saturation: Double = 0.0,
    val contrast: Double = 0.0,
    val exposure: Double = 0.0,
    val mode: String = "",
    val focusDistance: Double = 0.0,
    val aperture: Double = 0.0,
    val maxBlur: Double = 0.0,
)

fun gsxScene3DSpreadString(values: Map<String, Any?>, key: String, fallback: String): String {
    val value = values[key] ?: return fallback
    return value as? String ?: value.toString()
}

fun gsxScene3DSpreadFloat(values: Map<String, Any?>, key: String, fallback: Double): Double =
    when (val value = values[key]) {
        is Number -> value.toDouble()
        is String -> value.toDoubleOrNull() ?: fallback
        else -> fallback
    }

fun gsxScene3DSpreadInt(values: Map<String, Any?>, key: String, fallback: Int): Int =
    when (val value = values[key]) {
        is Number -> value.toInt()
        is String -> value.toIntOrNull() ?: fallback
        else -> fallback
    }

fun gsxScene3DSpreadBool(values: Map<String, Any?>, key: String, fallback: Boolean): Boolean =
    when (val value = values[key]) {
        is Boolean -> value
        is Number -> value.toInt() != 0
        is String -> when (value.trim().lowercase()) {
            "true", "1", "yes", "on" -> true
            "false", "0", "no", "off" -> false
            else -> fallback
        }
        else -> fallback
    }

data class GSXScene3DNode(
    val id: String,
    val tag: String,
    val kind: String = "",
    val color: String = "#8de1ff",
    val x: Double = 0.0,
    val y: Double = 0.0,
    val z: Double = 0.0,
    val width: Double = 1.0,
    val height: Double = 1.0,
    val depth: Double = 1.0,
    val count: Int = 0,
    val size: Double = 0.0,
)

@Composable
fun GSXScene3D(scene: GSXScene3DScene, modifier: Modifier = Modifier) {
    val aspect = (scene.width / max(scene.height, 1.0)).toFloat()
    Box(
        modifier = modifier
            .fillMaxWidth()
            .aspectRatio(aspect)
            .semantics { contentDescription = "Scene3D" }
            .testTag("Scene3D"),
    ) {
        Canvas(modifier = Modifier.fillMaxSize()) {
            drawRect(color = colorFromHex(scene.background), size = size)
            val renderableNodes = scene.nodes.filter { it.tag == "mesh" || it.tag == "model" || it.tag == "points" || it.tag == "instancedMesh" || it.tag == "computeParticles" }
            renderableNodes.forEachIndexed { index, node ->
                drawSceneNode(node, index, renderableNodes.size)
            }
        }
        scene.htmlOverlays.forEach { overlay ->
            BasicText(
                text = plainSceneHTMLText(overlay.html),
                modifier = Modifier
                    .align(Alignment.Center)
                    .offset {
                        IntOffset(
                            x = (overlay.x * 36.0 + overlay.offsetX).roundToInt(),
                            y = (-overlay.y * 36.0 + overlay.z * 4.0 + overlay.offsetY).roundToInt(),
                        )
                    }
                    .background(Color.Black.copy(alpha = 0.58f))
                    .padding(horizontal = 8.dp, vertical = 6.dp),
                style = TextStyle(color = Color.White.copy(alpha = overlay.opacity.coerceIn(0.0, 1.0).toFloat()), fontSize = 12.sp),
            )
        }
    }
}

private fun DrawScope.drawSceneNode(node: GSXScene3DNode, index: Int, total: Int) {
    val slotCount = max(total, 1)
    val slotWidth = size.width / (slotCount + 1)
    val center = Offset(
        x = slotWidth * (index + 1) + node.x.toFloat() * 24f,
        y = size.height * 0.5f - node.y.toFloat() * 24f + node.z.toFloat() * 4f,
    )
    val scale = minOf(size.width, size.height) * 0.18f
    val width = max(node.width.toFloat() * scale, 16f)
    val height = max(node.height.toFloat() * scale, 16f)
    val topLeft = Offset(center.x - width / 2f, center.y - height / 2f)
    val color = colorFromHex(node.color)

    when (node.tag) {
        "instancedMesh" -> {
            val count = max(node.count, 1)
            val instanceWidth = max(width * 0.42f, 10f)
            val instanceHeight = max(height * 0.42f, 10f)
            repeat(count) { i ->
                val offset = (i.toFloat() - (count - 1).toFloat() / 2f) * instanceWidth * 0.72f
                val rise = ((i % 2) * 2 - 1).toFloat() * instanceHeight * 0.18f
                val instanceTopLeft = Offset(center.x + offset - instanceWidth / 2f, center.y + rise - instanceHeight / 2f)
                val instanceSize = Size(instanceWidth, instanceHeight)
                drawRoundRect(color = color.copy(alpha = 0.84f), topLeft = instanceTopLeft, size = instanceSize, cornerRadius = CornerRadius(6f, 6f))
                drawRoundRect(color = Color.White.copy(alpha = 0.32f), topLeft = instanceTopLeft, size = instanceSize, cornerRadius = CornerRadius(6f, 6f), style = Stroke(width = 1f))
            }
        }
        "computeParticles" -> {
            val count = max(minOf(node.count, 48), 1)
            val radius = max(node.size.toFloat() * 8f, 2f)
            repeat(count) { i ->
                val angle = i.toFloat() * 0.62f
                val spiral = i.toFloat() / count.toFloat() * max(width, height) * 0.5f
                drawCircle(
                    color = color.copy(alpha = 0.72f),
                    radius = radius,
                    center = Offset(
                        x = center.x + cos(angle) * spiral,
                        y = center.y + sin(angle) * spiral * 0.6f,
                    ),
                )
            }
        }
        "points" -> {
            val count = max(node.count, 1)
            val radius = max(node.size.toFloat() * 8f, 3f)
            repeat(count) { i ->
                val offset = (i - count / 2) * radius * 2.4f
                drawCircle(color = color.copy(alpha = 0.9f), radius = radius, center = Offset(center.x + offset, center.y))
            }
        }
        "model" -> {
            drawOval(color = color.copy(alpha = 0.82f), topLeft = topLeft, size = Size(width, height))
            drawOval(color = Color.White.copy(alpha = 0.35f), topLeft = topLeft, size = Size(width, height), style = Stroke(width = 1f))
        }
        else -> {
            drawRoundRect(color = color.copy(alpha = 0.86f), topLeft = topLeft, size = Size(width, height), cornerRadius = CornerRadius(10f, 10f))
            drawRoundRect(color = Color.White.copy(alpha = 0.38f), topLeft = topLeft, size = Size(width, height), cornerRadius = CornerRadius(10f, 10f), style = Stroke(width = 1f))
        }
    }
}

private fun plainSceneHTMLText(html: String): String =
    html.replace(Regex("<[^>]+>"), " ")
        .replace(Regex("\\s+"), " ")
        .trim()

private fun colorFromHex(hex: String): Color {
    val cleaned = hex.trim().removePrefix("#")
    val value = cleaned.toLongOrNull(16) ?: return fallbackSceneColor()
    return when (cleaned.length) {
        6 -> Color(
            red = ((value shr 16) and 0xffL).toInt(),
            green = ((value shr 8) and 0xffL).toInt(),
            blue = (value and 0xffL).toInt(),
        )
        8 -> Color(
            red = ((value shr 16) and 0xffL).toInt(),
            green = ((value shr 8) and 0xffL).toInt(),
            blue = (value and 0xffL).toInt(),
            alpha = ((value shr 24) and 0xffL).toInt(),
        )
        else -> fallbackSceneColor()
    }
}

private fun fallbackSceneColor(): Color = Color(red = 16, green = 24, blue = 32)
