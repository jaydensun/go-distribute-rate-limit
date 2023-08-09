package main

import (
    "context"
    "flag"
    "fmt"
    "github.com/go-redis/redis_rate/v10"
    "github.com/redis/go-redis/v9"
    "os"
    "sync"
    "time"
)

func main() {
    var addr, password string
    flag.StringVar(&addr, "addr", "", "redis address, like localhost:6379")
    flag.StringVar(&password, "password", "", "redis password")
    flag.Parse()

    if addr == "" || password == "" {
        fmt.Println("invalid parameter")
        os.Exit(1)
    }

    ctx := context.Background()
    rdb := redis.NewClient(&redis.Options{
        Addr:     addr,
        Password: password,
    })
    _ = rdb.FlushDB(ctx).Err()

    limiter := redis_rate.NewLimiter(rdb)

    waitGroup := sync.WaitGroup{}
    start := time.Now()
    var doTimes []float64
    for i := 0; i < 500; i++ {
        go func() {
            waitGroup.Add(1)
            for {
                res, err := limiter.Allow(ctx, "project:123", redis_rate.PerSecond(50))
                if err != nil {
                    panic(err)
                }
                //fmt.Println("allowed", res.Allowed, "remaining", res.Remaining)
                // Output: allowed 1 remaining 9
                if res.Allowed > 0 {
                    rdb.Incr(ctx, "test")
                    doTimes = append(doTimes, time.Now().Sub(start).Seconds())
                } else {
                    time.Sleep(res.RetryAfter)
                }
                if time.Since(start) > time.Second*20 {
                    break
                }
            }
            waitGroup.Done()
        }()
    }
    waitGroup.Wait()
    fmt.Println(rdb.Get(ctx, "test"))
    fmt.Println(doTimes)
    startT := float64(0)
    var timeArray []float64
    for _, t := range doTimes {
        if t-startT > 1 {
            fmt.Println(len(timeArray))
            fmt.Println(timeArray)
            timeArray = make([]float64, 0, 100)
            startT = startT + 1
        } else {
            timeArray = append(timeArray, t)
        }
    }
}
