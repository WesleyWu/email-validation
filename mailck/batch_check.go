package mailck

import (
	"context"
	"github.com/gogf/gf/v2/container/gset"
	"github.com/gogf/gf/v2/frame/g"
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
	defer func() {
		message.Wg.Done()
	}()
	result, err := Check(message.FromEmail, message.CheckEmail)
	if err != nil {
		g.Log().Error(message.Ctx, err)
		result = MailserverError
	}
	retry := 0
	for result.IsValid() && retry < 5 {
		result, err = Check(message.FromEmail, message.CheckEmail)
		if err != nil {
			g.Log().Error(message.Ctx, err)
			result = MailserverError
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
	g.Log().Infof(ctx, "共 %d 条email需要验证", emailSet.Size())
	go func() {
		wg.Wait()
		close(results)
	}()
	count := 0
	for r := range results {
		resultSlice = append(resultSlice, r)
		count++
		if count%1000 == 0 {
			g.Log().Infof(ctx, "已验证 %d 条 email", count)
		}
	}
	g.Log().Infof(ctx, "验证完毕 %d 条email", emailSet.Size())

	return
}
