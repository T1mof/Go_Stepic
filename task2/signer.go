package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

func ExecutePipeline(jobs ...job) {
	if len(jobs) == 0 {
		return
	}

	channels := make([]chan interface{}, len(jobs)+1)
	for i := range channels {
		channels[i] = make(chan interface{}, 100)
	}

	var wg sync.WaitGroup

	for i, j := range jobs {
		wg.Add(1)
		go func(j job, in, out chan interface{}) {
			defer wg.Done()
			defer close(out)
			j(in, out)
		}(j, channels[i], channels[i+1])
	}
	close(channels[0])

	wg.Wait()
}

func SingleHash(in, out chan interface{}) {
	var mu sync.Mutex
	var wg sync.WaitGroup
	for num := range in {
		wg.Add(1)
		go func(num int, mu *sync.Mutex, wg *sync.WaitGroup) {
			defer wg.Done()
			n := strconv.Itoa(num)

			var crc1, crc2, md5Result string
			var wgg sync.WaitGroup

			wgg.Add(2)

			go func() {
				defer wgg.Done()
				crc1 = DataSignerCrc32(n)
			}()

			go func() {
				defer wgg.Done()
				mu.Lock()
				md5Result = DataSignerMd5(n)
				mu.Unlock()
				crc2 = DataSignerCrc32(md5Result)
			}()

			wgg.Wait()

			result := crc1 + "~" + crc2
			out <- result

		}(num.(int), &mu, &wg)
	}
	wg.Wait()
}

func MultiHash(in, out chan interface{}) {
	var wg sync.WaitGroup
	for num := range in {
		wg.Add(1)
		go func(num interface{}, wg *sync.WaitGroup) {
			defer wg.Done()
			var wgg sync.WaitGroup
			str := make([]string, 6)
			for th := 0; th < 6; th++ {
				wgg.Add(1)
				go func(str []string, th int, wgg *sync.WaitGroup) {
					defer wgg.Done()
					str[th] = DataSignerCrc32(strconv.Itoa(th) + num.(string))
				}(str, th, &wgg)
			}
			wgg.Wait()
			out <- strings.Join(str, "")
		}(num, &wg)
	}
	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	res := make([]string, 0)
	for i := range in {
		res = append(res, i.(string))
	}
	sort.Strings(res)
	out <- strings.Join(res, "_")
}
