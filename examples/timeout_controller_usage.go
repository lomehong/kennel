package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/lomehong/kennel/pkg/core"
	"github.com/lomehong/kennel/pkg/timeout"
)

// 本示例展示如何在AppFramework中使用超时控制器
func main() {
	// 创建应用程序实例
	app := core.NewApp("config.yaml")

	// 初始化应用程序
	if err := app.Init(); err != nil {
		fmt.Printf("初始化应用程序失败: %v\n", err)
		os.Exit(1)
	}

	// 获取超时控制器
	controller := app.GetTimeoutController()
	if controller == nil {
		fmt.Println("超时控制器未初始化")
		os.Exit(1)
	}

	fmt.Println("=== 超时控制器使用示例 ===")

	// 示例1: 使用WithTimeout执行函数
	fmt.Println("\n=== 示例1: 使用WithTimeout执行函数 ===")
	useWithTimeout(app)

	// 示例2: 使用ExecuteWithTimeout执行函数
	fmt.Println("\n=== 示例2: 使用ExecuteWithTimeout执行函数 ===")
	useExecuteWithTimeout(app)

	// 示例3: 使用IO操作超时控制
	fmt.Println("\n=== 示例3: 使用IO操作超时控制 ===")
	useIOTimeout(app)

	// 示例4: 使用网络操作超时控制
	fmt.Println("\n=== 示例4: 使用网络操作超时控制 ===")
	useNetworkTimeout(app)

	// 示例5: 使用重试机制
	fmt.Println("\n=== 示例5: 使用重试机制 ===")
	useRetryMechanism(app)

	// 停止应用程序
	app.Stop()
	fmt.Println("\n应用程序已停止")
}

// 使用WithTimeout执行函数
func useWithTimeout(app *core.App) {
	// 使用WithTimeout执行一个正常完成的函数
	err := app.WithTimeout(timeout.OperationTypeIO, "normal-operation", "正常操作", 5*time.Second, func(ctx context.Context) error {
		fmt.Println("执行正常操作...")
		time.Sleep(1 * time.Second)
		fmt.Println("正常操作完成")
		return nil
	})

	if err != nil {
		fmt.Printf("正常操作失败: %v\n", err)
	} else {
		fmt.Println("正常操作成功")
	}

	// 使用WithTimeout执行一个会超时的函数
	err = app.WithTimeout(timeout.OperationTypeIO, "timeout-operation", "超时操作", 2*time.Second, func(ctx context.Context) error {
		fmt.Println("执行可能超时的操作...")

		select {
		case <-time.After(5 * time.Second):
			fmt.Println("操作完成，但已超时")
			return nil
		case <-ctx.Done():
			fmt.Printf("操作被取消: %v\n", ctx.Err())
			return ctx.Err()
		}
	})

	if err != nil {
		fmt.Printf("超时操作失败: %v\n", err)
	} else {
		fmt.Println("超时操作成功")
	}
}

// 使用ExecuteWithTimeout执行函数
func useExecuteWithTimeout(app *core.App) {
	// 使用ExecuteWithTimeout执行一个正常完成的函数
	err := app.ExecuteWithTimeout(timeout.OperationTypeNetwork, "HTTP请求", 5*time.Second, func(ctx context.Context) error {
		fmt.Println("执行HTTP请求...")

		// 创建带有上下文的HTTP请求
		req, err := http.NewRequestWithContext(ctx, "GET", "https://example.com", nil)
		if err != nil {
			return err
		}

		// 执行请求
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// 读取响应
		_, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		fmt.Println("HTTP请求完成")
		return nil
	})

	if err != nil {
		fmt.Printf("HTTP请求失败: %v\n", err)
	} else {
		fmt.Println("HTTP请求成功")
	}
}

// 使用IO操作超时控制
func useIOTimeout(app *core.App) {
	// 创建临时文件
	file, err := ioutil.TempFile("", "timeout-example-*.txt")
	if err != nil {
		fmt.Printf("创建临时文件失败: %v\n", err)
		return
	}
	defer os.Remove(file.Name())

	// 写入数据
	data := []byte("Hello, Timeout Control!")
	err = app.IOOperation("写入文件", 5*time.Second, func() error {
		_, err := file.Write(data)
		return err
	})

	if err != nil {
		fmt.Printf("写入文件失败: %v\n", err)
	} else {
		fmt.Println("写入文件成功")
	}

	// 重置文件指针
	_, err = file.Seek(0, 0)
	if err != nil {
		fmt.Printf("重置文件指针失败: %v\n", err)
		return
	}

	// 读取数据
	buffer := make([]byte, 100)
	n, err := app.ReadWithTimeout(file, buffer, 5*time.Second)
	if err != nil {
		fmt.Printf("读取文件失败: %v\n", err)
	} else {
		fmt.Printf("读取文件成功: %s\n", buffer[:n])
	}

	// 关闭文件
	err = app.CloseWithTimeout(file, 5*time.Second)
	if err != nil {
		fmt.Printf("关闭文件失败: %v\n", err)
	} else {
		fmt.Println("关闭文件成功")
	}
}

// 使用网络操作超时控制
func useNetworkTimeout(app *core.App) {
	// 使用超时执行HTTP请求
	err := app.ExecuteWithTimeout(timeout.OperationTypeNetwork, "带超时的HTTP请求", 3*time.Second, func(ctx context.Context) error {
		fmt.Println("执行带超时的HTTP请求...")

		// 创建带有上下文的HTTP请求
		req, err := http.NewRequestWithContext(ctx, "GET", "https://example.com", nil)
		if err != nil {
			return err
		}

		// 使用带有超时的HTTP客户端
		client := &http.Client{
			Timeout: 2 * time.Second, // 客户端级别的超时
		}

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// 读取响应
		_, err = ioutil.ReadAll(resp.Body)
		return err
	})

	if err != nil {
		fmt.Printf("带超时的HTTP请求失败: %v\n", err)
	} else {
		fmt.Println("带超时的HTTP请求成功")
	}
}

// 使用重试机制
func useRetryMechanism(app *core.App) {
	// 模拟不稳定的操作
	attempt := 0
	err := app.RetryWithTimeout("不稳定的操作", 3, 1*time.Second, 10*time.Second, func() error {
		attempt++
		fmt.Printf("尝试执行操作，第%d次\n", attempt)

		// 模拟前两次失败，第三次成功
		if attempt < 3 {
			fmt.Println("操作失败，将重试")
			return fmt.Errorf("模拟失败，尝试次数: %d", attempt)
		}

		fmt.Println("操作成功")
		return nil
	})

	if err != nil {
		fmt.Printf("重试操作最终失败: %v\n", err)
	} else {
		fmt.Printf("重试操作最终成功，尝试次数: %d\n", attempt)
	}
}
