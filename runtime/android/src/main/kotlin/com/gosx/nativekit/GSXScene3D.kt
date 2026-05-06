package com.gosx.nativekit

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.runtime.Composable
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
import kotlin.math.max

data class GSXScene3DScene(
    val width: Double = 640.0,
    val height: Double = 360.0,
    val background: String = "#101820",
    val nodes: List<GSXScene3DNode> = emptyList(),
)

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
    Canvas(
        modifier = modifier
            .fillMaxWidth()
            .aspectRatio(aspect)
            .semantics { contentDescription = "Scene3D" }
            .testTag("Scene3D"),
    ) {
        drawRect(color = colorFromHex(scene.background), size = size)
        val renderableNodes = scene.nodes.filter { it.tag == "mesh" || it.tag == "model" || it.tag == "points" }
        renderableNodes.forEachIndexed { index, node ->
            drawSceneNode(node, index, renderableNodes.size)
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
