package mailck

import (
	"context"
	"github.com/gogf/gf/v2/container/gset"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/panjf2000/ants/v2"
	"sync"
)

var (
	pool *ants.PoolWithFunc
)

type CheckTask struct {
	FromEmail  string `json:"-"`
	CheckEmail string `json:"-"`
	Ctx        context.Context
	Results    chan *CheckResult
	Wg         *sync.WaitGroup `json:"-"`
}

type CheckResult struct {
	CheckEmail string
	Result
}

func init() {
	pool, _ = ants.NewPoolWithFunc(500, func(i interface{}) {
		handleAll(i)
	})
}

func handleAll(i interface{}) {
	message, _ := i.(*CheckTask)
	defer message.Wg.Done()
	result, err := Check(message.FromEmail, message.CheckEmail)
	if err != nil {
		if pos := gstr.Pos(err.Error(), "550"); pos == 0 {
			result = MailserverError
		} else {
			g.Log().Error(message.Ctx, err)
			return
		}
	}
	retry := 0
	for result.IsValid() && retry < 5 {
		result, err = Check(message.FromEmail, message.CheckEmail)
		if err != nil {
			if pos := gstr.Pos(err.Error(), "550"); pos == 0 {
				result = MailserverError
			} else {
				g.Log().Error(message.Ctx, err)
				return
			}
		}
		retry++
	}
	message.Results <- &CheckResult{
		CheckEmail: message.CheckEmail,
		Result:     result,
	}
}

func BatchCheck(ctx context.Context, emailSet *gset.StrSet) (resultSlice []*CheckResult, err error) {
	var (
		wg      sync.WaitGroup
		results = make(chan *CheckResult)
	)
	fromEmail := g.Cfg().MustGet(ctx, "settings.fromEmail", "noreply@cartx.ai").String()
	emailSet.Iterator(func(email string) bool {
		wg.Add(1)
		err = pool.Invoke(&CheckTask{
			FromEmail:  fromEmail,
			CheckEmail: email,
			Ctx:        ctx,
			Results:    results,
			Wg:         &wg,
		})
		if err != nil {
			return false
		}
		return true
	})
	if err != nil {
		return
	}
	go func() {
		wg.Wait()
		close(results)
	}()
	for r := range results {
		resultSlice = append(resultSlice, r)
	}
	return
}
