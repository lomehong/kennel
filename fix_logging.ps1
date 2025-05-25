# PowerShell脚本：批量替换日志相关的导入和类型

# 定义需要替换的文件列表
$files = @(
    "app\control\pkg\ai\mcp\model_client.go",
    "app\control\pkg\ai\mcp\server.go", 
    "app\control\pkg\ai\mcp\tools.go",
    "app\control\pkg\ai\mcp\tool_impl.go",
    "app\control\pkg\ai\tools.go",
    "app\control\pkg\control\command.go",
    "app\control\pkg\control\process.go",
    "app\device\network.go",
    "app\device\usb.go"
)

foreach ($file in $files) {
    if (Test-Path $file) {
        Write-Host "处理文件: $file"
        
        # 读取文件内容
        $content = Get-Content $file -Raw
        
        # 替换导入语句
        $content = $content -replace 'sdk "github.com/lomehong/kennel/pkg/sdk/go"', '"github.com/lomehong/kennel/pkg/logging"'
        $content = $content -replace '"github.com/lomehong/kennel/pkg/sdk/go"', '"github.com/lomehong/kennel/pkg/logging"'
        
        # 替换类型声明
        $content = $content -replace 'logger\s+sdk\.Logger', 'logger logging.Logger'
        $content = $content -replace 'Logger\s+sdk\.Logger', 'Logger logging.Logger'
        
        # 替换函数参数
        $content = $content -replace 'logger\s+sdk\.Logger\)', 'logger logging.Logger)'
        $content = $content -replace 'Logger\s+sdk\.Logger\)', 'Logger logging.Logger)'
        
        # 替换函数签名中的参数
        $content = $content -replace '(\w+)\s+sdk\.Logger', '$1 logging.Logger'
        
        # 写回文件
        $content | Set-Content $file -NoNewline
        
        Write-Host "完成处理: $file"
    } else {
        Write-Host "文件不存在: $file"
    }
}

Write-Host "批量替换完成！"
