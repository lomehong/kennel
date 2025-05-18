package concurrency

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPipeline(t *testing.T) {
	// 创建输入通道
	in := make(chan interface{})
	
	// 创建处理阶段
	stage1 := func(in <-chan interface{}) <-chan interface{} {
		out := make(chan interface{})
		go func() {
			defer close(out)
			for v := range in {
				out <- v.(int) * 2
			}
		}()
		return out
	}
	
	stage2 := func(in <-chan interface{}) <-chan interface{} {
		out := make(chan interface{})
		go func() {
			defer close(out)
			for v := range in {
				out <- v.(int) + 1
			}
		}()
		return out
	}
	
	// 创建管道
	out := Pipeline(in, stage1, stage2)
	
	// 发送数据
	go func() {
		in <- 1
		in <- 2
		in <- 3
		close(in)
	}()
	
	// 收集结果
	var results []int
	for v := range out {
		results = append(results, v.(int))
	}
	
	// 验证结果
	assert.Equal(t, []int{3, 5, 7}, results)
}

func TestFanOut(t *testing.T) {
	// 创建输入通道
	in := make(chan interface{})
	
	// 创建worker函数
	worker := func(v interface{}) interface{} {
		return v.(int) * 2
	}
	
	// 创建扇出
	out := FanOut(in, 3, worker)
	
	// 发送数据
	go func() {
		for i := 1; i <= 10; i++ {
			in <- i
		}
		close(in)
	}()
	
	// 收集结果
	var results []int
	for v := range out {
		results = append(results, v.(int))
	}
	
	// 验证结果
	assert.Len(t, results, 10)
	assert.ElementsMatch(t, []int{2, 4, 6, 8, 10, 12, 14, 16, 18, 20}, results)
}

func TestFanIn(t *testing.T) {
	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// 创建输入通道
	ch1 := make(chan interface{})
	ch2 := make(chan interface{})
	ch3 := make(chan interface{})
	
	// 创建扇入
	out := FanIn(ctx, ch1, ch2, ch3)
	
	// 发送数据
	go func() {
		ch1 <- 1
		ch1 <- 2
		close(ch1)
	}()
	
	go func() {
		ch2 <- 3
		ch2 <- 4
		close(ch2)
	}()
	
	go func() {
		ch3 <- 5
		ch3 <- 6
		close(ch3)
	}()
	
	// 收集结果
	var results []int
	for v := range out {
		results = append(results, v.(int))
	}
	
	// 验证结果
	assert.Len(t, results, 6)
	assert.ElementsMatch(t, []int{1, 2, 3, 4, 5, 6}, results)
}

func TestBatch(t *testing.T) {
	// 创建输入通道
	in := make(chan interface{})
	
	// 创建批处理
	out := Batch(in, 3)
	
	// 发送数据
	go func() {
		for i := 1; i <= 10; i++ {
			in <- i
		}
		close(in)
	}()
	
	// 收集结果
	var results [][]interface{}
	for batch := range out {
		results = append(results, batch)
	}
	
	// 验证结果
	assert.Len(t, results, 4)
	assert.Equal(t, []interface{}{1, 2, 3}, results[0])
	assert.Equal(t, []interface{}{4, 5, 6}, results[1])
	assert.Equal(t, []interface{}{7, 8, 9}, results[2])
	assert.Equal(t, []interface{}{10}, results[3])
}

func TestThrottle(t *testing.T) {
	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// 创建输入通道
	in := make(chan interface{})
	
	// 创建处理函数
	fn := func(v interface{}) interface{} {
		time.Sleep(100 * time.Millisecond)
		return v.(int) * 2
	}
	
	// 创建限流
	out := Throttle(ctx, in, 3, fn)
	
	// 记录开始时间
	start := time.Now()
	
	// 发送数据
	go func() {
		for i := 1; i <= 6; i++ {
			in <- i
		}
		close(in)
	}()
	
	// 收集结果
	var results []int
	for v := range out {
		results = append(results, v.(int))
	}
	
	// 验证结果
	duration := time.Since(start)
	assert.Len(t, results, 6)
	assert.ElementsMatch(t, []int{2, 4, 6, 8, 10, 12}, results)
	
	// 验证限流效果（应该至少需要200毫秒，因为有6个任务，并发度为3）
	assert.True(t, duration >= 200*time.Millisecond)
}

func TestMap(t *testing.T) {
	// 创建输入通道
	in := make(chan interface{})
	
	// 创建映射函数
	fn := func(v interface{}) interface{} {
		return v.(int) * 2
	}
	
	// 创建映射
	out := Map(in, fn)
	
	// 发送数据
	go func() {
		for i := 1; i <= 5; i++ {
			in <- i
		}
		close(in)
	}()
	
	// 收集结果
	var results []int
	for v := range out {
		results = append(results, v.(int))
	}
	
	// 验证结果
	assert.Equal(t, []int{2, 4, 6, 8, 10}, results)
}

func TestFilter(t *testing.T) {
	// 创建输入通道
	in := make(chan interface{})
	
	// 创建过滤函数
	fn := func(v interface{}) bool {
		return v.(int)%2 == 0
	}
	
	// 创建过滤
	out := Filter(in, fn)
	
	// 发送数据
	go func() {
		for i := 1; i <= 10; i++ {
			in <- i
		}
		close(in)
	}()
	
	// 收集结果
	var results []int
	for v := range out {
		results = append(results, v.(int))
	}
	
	// 验证结果
	assert.Equal(t, []int{2, 4, 6, 8, 10}, results)
}

func TestReduce(t *testing.T) {
	// 创建输入通道
	in := make(chan interface{})
	
	// 创建归约函数
	fn := func(acc, v interface{}) interface{} {
		return acc.(int) + v.(int)
	}
	
	// 发送数据
	go func() {
		for i := 1; i <= 5; i++ {
			in <- i
		}
		close(in)
	}()
	
	// 执行归约
	result := Reduce(in, 0, fn)
	
	// 验证结果
	assert.Equal(t, 15, result)
}

func TestForEach(t *testing.T) {
	// 创建输入通道
	in := make(chan interface{})
	
	// 创建结果切片
	var results []int
	var mu sync.Mutex
	
	// 创建处理函数
	fn := func(v interface{}) {
		mu.Lock()
		defer mu.Unlock()
		results = append(results, v.(int))
	}
	
	// 发送数据
	go func() {
		for i := 1; i <= 5; i++ {
			in <- i
		}
		close(in)
	}()
	
	// 执行ForEach
	ForEach(in, fn)
	
	// 验证结果
	assert.Len(t, results, 5)
	assert.ElementsMatch(t, []int{1, 2, 3, 4, 5}, results)
}

func TestTee(t *testing.T) {
	// 创建输入通道
	in := make(chan interface{})
	
	// 创建Tee
	outs := Tee(in, 3)
	
	// 发送数据
	go func() {
		for i := 1; i <= 3; i++ {
			in <- i
		}
		close(in)
	}()
	
	// 收集结果
	var results [][]int
	var wg sync.WaitGroup
	var mu sync.Mutex
	
	for i, out := range outs {
		wg.Add(1)
		go func(i int, ch <-chan interface{}) {
			defer wg.Done()
			var channelResults []int
			for v := range ch {
				channelResults = append(channelResults, v.(int))
			}
			mu.Lock()
			results = append(results, channelResults)
			mu.Unlock()
		}(i, out)
	}
	
	wg.Wait()
	
	// 验证结果
	assert.Len(t, results, 3)
	for _, result := range results {
		assert.Equal(t, []int{1, 2, 3}, result)
	}
}

func TestOrDone(t *testing.T) {
	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	
	// 创建输入通道
	in := make(chan interface{})
	
	// 创建OrDone
	out := OrDone(ctx, in)
	
	// 发送数据
	go func() {
		for i := 1; i <= 10; i++ {
			in <- i
			time.Sleep(50 * time.Millisecond)
		}
		close(in)
	}()
	
	// 收集结果
	var results []int
	
	// 读取前5个值，然后取消上下文
	for i := 0; i < 5; i++ {
		v := <-out
		results = append(results, v.(int))
	}
	
	// 取消上下文
	cancel()
	
	// 等待一段时间，确保通道关闭
	time.Sleep(100 * time.Millisecond)
	
	// 验证结果
	assert.Len(t, results, 5)
	assert.Equal(t, []int{1, 2, 3, 4, 5}, results)
}
