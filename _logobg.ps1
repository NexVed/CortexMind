
Add-Type -AssemblyName System.Drawing
$srcPath = 'C:\Users\u9780\Cortex\ui\public\logowithname.png'
$outPath = 'C:\Users\u9780\Cortex\ui\public\logowithname-readme.png'
$src = [System.Drawing.Image]::FromFile($srcPath)
$padX = 60
$padY = 40
$radius = 28
$w = $src.Width + 2 * $padX
$h = $src.Height + 2 * $padY
$bmp = New-Object System.Drawing.Bitmap($w, $h)
$g = [System.Drawing.Graphics]::FromImage($bmp)
$g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
$g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
$g.Clear([System.Drawing.Color]::Transparent)
$path = New-Object System.Drawing.Drawing2D.GraphicsPath
$d = 2 * $radius
$path.AddArc(0, 0, $d, $d, 180, 90)
$path.AddArc($w - $d, 0, $d, $d, 270, 90)
$path.AddArc($w - $d, $h - $d, $d, $d, 0, 90)
$path.AddArc(0, $h - $d, $d, $d, 90, 90)
$path.CloseFigure()
$brush = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::White)
$g.FillPath($brush, $path)
$g.DrawImage($src, $padX, $padY, $src.Width, $src.Height)
$bmp.Save($outPath, [System.Drawing.Imaging.ImageFormat]::Png)
$g.Dispose()
$bmp.Dispose()
$src.Dispose()
$brush.Dispose()
$path.Dispose()
Write-Output ("Wrote " + $outPath + " " + $w + "x" + $h)
