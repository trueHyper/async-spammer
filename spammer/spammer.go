package spammer 

import (
	"fmt"
	"log"
	"sort"
	"sync"
)

const (
	ExpectedNumOfAlias    = 16
	ExpectedNumOfMessages = 32
)

type spammer struct {
	key   MsgID
	value bool
}

func RunPipeline(cmds ...cmd) {

	in := make(chan interface{})
	defer close(in)

	var wg sync.WaitGroup

	for _, fn := range cmds {
		wg.Add(1)

		out := make(chan interface{})

		go func(fn cmd, chIn, chOut chan interface{}) {
			defer func() {
				close(chOut)
				wg.Done()
			}()
			fn(chIn, chOut)
		}(fn, in, out)

		in = out
	}

	wg.Wait()
}

func SelectUsers(in, out chan interface{}) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	alias := make(map[uint64]struct{}, ExpectedNumOfAlias)

	for rawEmail := range in {

		wg.Add(1)

		go func(rawEmail interface{}, wg *sync.WaitGroup) {
			defer wg.Done()

			email, ok := rawEmail.(string)
			if !ok {
				log.Printf("SelectUser: expected string, got %T\n", email)
				return
			}
			user := GetUser(email)

			mu.Lock() /* замок на мапу алиасов для чтения и записи */
			_, found := alias[user.ID]
			if !found {
				alias[user.ID] = struct{}{}
			}
			mu.Unlock()

			if !found {
				out <- user
			}
		}(rawEmail, &wg)
	}

	wg.Wait()
}

func SelectMessages(in, out chan interface{}) {
	var wg sync.WaitGroup
	batch := make([]User, 0, GetMessagesMaxUsersBatch)

	getMsgs := func(wg *sync.WaitGroup, users ...User) {
		defer wg.Done()

		msgs, err := GetMessages(users...)
		if err != nil {
			log.Println("GetMessages err:", err)
			return
		}

		for _, msg := range msgs {
			out <- msg
		}
	}

	for val := range in {
		user, ok := val.(User)
		if !ok {
			log.Printf("expected User, got %T\n", val)
			continue
		}

		batch = append(batch, user)

		if len(batch) == GetMessagesMaxUsersBatch {
			wg.Add(1)
			go getMsgs(&wg, batch...)
			batch = nil
		}
	}

	if len(batch) > 0 {
		wg.Add(1)
		go getMsgs(&wg, batch...)
	}

	wg.Wait()
}

func CheckSpam(in, out chan interface{}) {

	workerInput := make(chan MsgID, HasSpamMaxAsyncRequests)

	var wg sync.WaitGroup

	for i := 0; i < HasSpamMaxAsyncRequests; i++ {
		wg.Add(1)

		go func(workerInput <-chan MsgID) {
			defer wg.Done()

			for msgid := range workerInput {
				result, err := HasSpam(msgid)
				if err != nil {
					log.Println("HasSpam err:", err)
					continue
				}
				out <- MsgData{ID: msgid, HasSpam: result}
			}
		}(workerInput)
	}

	for msg := range in {
		if m, ok := msg.(MsgID); ok {
			workerInput <- m
		} else {
			log.Printf("expected MsgID, got %T\n", m)
		}
	}

	close(workerInput)
	wg.Wait()
}

func CombineResults(in, out chan interface{}) {

	msgs := make([]spammer, 0, ExpectedNumOfMessages)

	for i := range in {
		if msg, ok := i.(MsgData); ok {
			msgs = append(msgs, spammer{key: msg.ID, value: msg.HasSpam})
		} else {
			log.Printf("expected MsgID, got %T\n", msg)
		}
	}

	sort.Slice(msgs, func(i, j int) bool {
		if msgs[i].value == msgs[j].value {
			return msgs[i].key < msgs[j].key
		}
		return msgs[i].value
	})

	for i := 0; i < len(msgs); i++ {
		out <- fmt.Sprintf("%v %d", msgs[i].value, msgs[i].key)
	}
}
