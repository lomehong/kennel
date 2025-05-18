package concurrency

import (
	"context"
	"sync"
)

// Pipeline 创建处理管道
func Pipeline(in <-chan interface{}, stages ...func(<-chan interface{}) <-chan interface{}) <-chan interface{} {
	out := in
	for _, stage := range stages {
		out = stage(out)
	}
	return out
}

// FanOut 将任务分发给多个worker并行处理
func FanOut(in <-chan interface{}, n int, worker func(interface{}) interface{}) <-chan interface{} {
	out := make(chan interface{})
	
	wg := sync.WaitGroup{}
	wg.Add(n)
	
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			for item := range in {
				out <- worker(item)
			}
		}()
	}
	
	go func() {
		wg.Wait()
		close(out)
	}()
	
	return out
}

// FanIn 合并多个通道的输出到一个通道
func FanIn(ctx context.Context, channels ...<-chan interface{}) <-chan interface{} {
	out := make(chan interface{})
	var wg sync.WaitGroup
	
	// 为每个输入通道启动一个goroutine
	wg.Add(len(channels))
	for _, c := range channels {
		go func(ch <-chan interface{}) {
			defer wg.Done()
			for {
				select {
				case v, ok := <-ch:
					if !ok {
						return
					}
					select {
					case out <- v:
					case <-ctx.Done():
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}(c)
	}
	
	// 当所有输入通道关闭时，关闭输出通道
	go func() {
		wg.Wait()
		close(out)
	}()
	
	return out
}

// Batch 将输入通道的项目分批处理
func Batch(in <-chan interface{}, batchSize int) <-chan []interface{} {
	out := make(chan []interface{})
	
	go func() {
		defer close(out)
		
		batch := make([]interface{}, 0, batchSize)
		for item := range in {
			batch = append(batch, item)
			
			if len(batch) >= batchSize {
				// 创建一个新的切片，避免数据竞争
				b := make([]interface{}, len(batch))
				copy(b, batch)
				out <- b
				batch = make([]interface{}, 0, batchSize)
			}
		}
		
		// 发送最后一个批次（如果有）
		if len(batch) > 0 {
			out <- batch
		}
	}()
	
	return out
}

// Throttle 限制并发执行的数量
func Throttle(ctx context.Context, in <-chan interface{}, limit int, fn func(interface{}) interface{}) <-chan interface{} {
	out := make(chan interface{})
	
	// 创建信号量通道
	sem := make(chan struct{}, limit)
	
	go func() {
		defer close(out)
		
		for item := range in {
			// 如果上下文已取消，退出
			if ctx.Err() != nil {
				return
			}
			
			// 获取信号量
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			
			// 处理项目
			go func(i interface{}) {
				defer func() { <-sem }() // 释放信号量
				
				result := fn(i)
				
				// 发送结果
				select {
				case out <- result:
				case <-ctx.Done():
				}
			}(item)
		}
		
		// 等待所有goroutine完成
		for i := 0; i < limit; i++ {
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
		}
	}()
	
	return out
}

// Map 对输入通道的每个项目应用函数
func Map(in <-chan interface{}, fn func(interface{}) interface{}) <-chan interface{} {
	out := make(chan interface{})
	
	go func() {
		defer close(out)
		for item := range in {
			out <- fn(item)
		}
	}()
	
	return out
}

// Filter 过滤输入通道中的项目
func Filter(in <-chan interface{}, fn func(interface{}) bool) <-chan interface{} {
	out := make(chan interface{})
	
	go func() {
		defer close(out)
		for item := range in {
			if fn(item) {
				out <- item
			}
		}
	}()
	
	return out
}

// Reduce 将输入通道中的项目归约为单个结果
func Reduce(in <-chan interface{}, initialValue interface{}, fn func(interface{}, interface{}) interface{}) interface{} {
	result := initialValue
	for item := range in {
		result = fn(result, item)
	}
	return result
}

// ForEach 对输入通道中的每个项目执行函数
func ForEach(in <-chan interface{}, fn func(interface{})) {
	for item := range in {
		fn(item)
	}
}

// Merge 合并多个通道的输出到一个通道（不保证顺序）
func Merge(channels ...<-chan interface{}) <-chan interface{} {
	out := make(chan interface{})
	var wg sync.WaitGroup
	
	// 为每个输入通道启动一个goroutine
	wg.Add(len(channels))
	for _, c := range channels {
		go func(ch <-chan interface{}) {
			defer wg.Done()
			for item := range ch {
				out <- item
			}
		}(c)
	}
	
	// 当所有输入通道关闭时，关闭输出通道
	go func() {
		wg.Wait()
		close(out)
	}()
	
	return out
}

// Tee 将输入通道的项目复制到多个输出通道
func Tee(in <-chan interface{}, n int) []<-chan interface{} {
	outs := make([]chan interface{}, n)
	for i := 0; i < n; i++ {
		outs[i] = make(chan interface{})
	}
	
	go func() {
		defer func() {
			for _, out := range outs {
				close(out)
			}
		}()
		
		for item := range in {
			for _, out := range outs {
				out <- item
			}
		}
	}()
	
	// 转换为只读通道
	results := make([]<-chan interface{}, n)
	for i, out := range outs {
		results[i] = out
	}
	
	return results
}

// OrDone 在上下文取消时关闭通道
func OrDone(ctx context.Context, in <-chan interface{}) <-chan interface{} {
	out := make(chan interface{})
	
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-in:
				if !ok {
					return
				}
				select {
				case out <- v:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	
	return out
}

// Bridge 将多个通道的通道扁平化为单个通道
func Bridge(ctx context.Context, chanStream <-chan <-chan interface{}) <-chan interface{} {
	out := make(chan interface{})
	
	go func() {
		defer close(out)
		
		for {
			var stream <-chan interface{}
			select {
			case maybeStream, ok := <-chanStream:
				if !ok {
					return
				}
				stream = maybeStream
			case <-ctx.Done():
				return
			}
			
			for item := range OrDone(ctx, stream) {
				select {
				case out <- item:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	
	return out
}
