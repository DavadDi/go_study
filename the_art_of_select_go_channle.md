# Golang Channle

## 1. 队列满则保存到backend

如果当前的channel的memoryChan已经满了，则默认写入到backend的缓存中

	func (c *Channel) put(m *Message) error {
		select {
		case c.memoryMsgChan <- m: // 当前缓存队列未满，则放入
		default:                   // 如果队列满，则保存到backend
			b := bufferPoolGet()                         
			err := writeMessageToBackend(b, m, c.backend)
			bufferPoolPut(b)
			c.ctx.nsqd.SetHealth(err)
			if err != nil {
				c.ctx.nsqd.logf("CHANNEL(%s) ERROR: failed to write message to backend - %s",
					c.name, err)
				return err
			}
		}
		return nil
	}
	
## 2. Polling Channle

	select {
	case <-exit:
		fmt.Println("It's done")

	default:
		// do nothing
	}
	
## 3. Polling Timeout

	tick := time.Tick(1*time.Second)
	select {
	case <-tick:
		fmt.Println("Timeout")

	case <- exit:
		fmt.Println("Exit")
		
	// do nothing
	}

	
## 4. 通过Channel实现随机概率分发

	select {
			case b := <-backendMsgChan:
			if sampleRate > 0 && rand.Int31n(100) > sampleRate {
				continue
			} 
	}
	
## 5. A trick

	ch := make(chan int, 1)
	
	for i := 0; i < 10; i++ {
		select {
		case x:= <-ch:
			fmt.Println(x) // 0 2 4 6 8

		case ch<-i:
		}
	}
	
## 6. Range on Closed Channel

  channel 关闭会导致range返回

	 queue := make(chan string, 2)
    queue <- "one"
    queue <- "two"
    close(queue)

    for elem := range queue {
        fmt.Println(elem)
    }
    
    
## 7. Check Closed Channel
 
 
   	queue := make(chan int, 1)
   	
	value, ok := <-queue
	if !ok {
		fmt.Println("queue is closed")
	}
	
## 8. Select skip nil Channel

	for {
	    select {
	    case x, ok := <-ch: // if ch is nil, just skip
	        fmt.Println("ch1", x, ok)
	        if !ok {
	            ch = nil
	        }
	}