package org.cloudreve.android.ui.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color

private val LightColors = lightColorScheme(
    primary = Color(0xFF4A6FA5),
    secondary = Color(0xFF6B8BB8),
    surface = Color(0xFFFAFBFD),
)

private val DarkColors = darkColorScheme(
    primary = Color(0xFF9DB8DC),
    secondary = Color(0xFF8AA4CC),
    surface = Color(0xFF121417),
)

@Composable
fun CloudreveTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = if (isSystemInDarkTheme()) DarkColors else LightColors,
        content = content,
    )
}
