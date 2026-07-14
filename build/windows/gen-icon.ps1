# Generates build/windows/icon.ico — the CortexMind logo mark centered on a
# white background (so the Windows app icon reads well on any taskbar/theme).
param(
    [string]$Source = (Join-Path $PSScriptRoot '..\..\ui\public\logo.png'),
    [string]$Out    = (Join-Path $PSScriptRoot 'icon.ico')
)

Add-Type -AssemblyName System.Drawing

$src = [System.Drawing.Image]::FromFile((Resolve-Path $Source))
$size = 256
$bmp = New-Object System.Drawing.Bitmap($size, $size, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
$g = [System.Drawing.Graphics]::FromImage($bmp)
$g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
$g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
$g.Clear([System.Drawing.Color]::White)

# Fit the logo within a padded square, centered.
$pad = 30
$avail = $size - 2 * $pad
$scale = [Math]::Min($avail / $src.Width, $avail / $src.Height)
$w = [int]($src.Width * $scale)
$h = [int]($src.Height * $scale)
$x = [int](($size - $w) / 2)
$y = [int](($size - $h) / 2)
$g.DrawImage($src, $x, $y, $w, $h)

# Encode the canvas as PNG in memory.
$ms = New-Object System.IO.MemoryStream
$bmp.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png)
$png = $ms.ToArray()

# Wrap the PNG in a single-image ICO (Windows Vista+ PNG icon format).
$icoStream = New-Object System.IO.MemoryStream
$bw = New-Object System.IO.BinaryWriter($icoStream)
$bw.Write([UInt16]0)             # reserved
$bw.Write([UInt16]1)             # type = icon
$bw.Write([UInt16]1)             # image count
$bw.Write([Byte]0)               # width  (0 => 256)
$bw.Write([Byte]0)               # height (0 => 256)
$bw.Write([Byte]0)               # palette colors
$bw.Write([Byte]0)               # reserved
$bw.Write([UInt16]1)             # color planes
$bw.Write([UInt16]32)            # bits per pixel
$bw.Write([UInt32]$png.Length)   # image data size
$bw.Write([UInt32]22)            # offset to image data (6 + 16)
$bw.Write($png)
$bw.Flush()

[System.IO.File]::WriteAllBytes((Join-Path $PSScriptRoot 'icon.ico'), $icoStream.ToArray())

$g.Dispose(); $bmp.Dispose(); $src.Dispose(); $bw.Dispose()
Write-Output ("icon.ico written (" + $png.Length + " bytes PNG payload)")
