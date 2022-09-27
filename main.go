package main

import (
	"bufio"
	"context"
	"github.com/WesleyWu/email-validation/mailck"
	"github.com/gogf/gf/v2/container/gset"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/text/gstr"
	"os"
)

func main() {
	command := gcmd.Command{
		Name: "Email batch validator",
		Func: BatchCheckByFile,
	}
	command.Run(gctx.New())
}

func BatchCheckByFile(ctx context.Context, parser *gcmd.Parser) error {
	input := parser.GetOpt("input").String()
	output := parser.GetOpt("outout").String()

	if g.IsEmpty(input) {
		return gerror.New("必须指定 input 文件")
	}

	inputFile, err := os.Open(input)
	if err != nil {
		return gerror.Newf("Cannot open inputFile: %s, err: [%v]", input, err)
	}
	defer inputFile.Close()

	scanner := bufio.NewScanner(inputFile)
	emailSet := gset.NewStrSet()
	for scanner.Scan() {
		line := gstr.Trim(gstr.ToLower(scanner.Text()))
		if !g.IsEmpty(line) {
			emailSet.Add(line)
		}
	}

	results, err := mailck.BatchCheck(ctx, emailSet)
	if err != nil {
		return err
	}

	if g.IsEmpty(output) {
		output = input + ".out"
	}
	outputFile, err := os.Create(output)
	defer outputFile.Close()
	writer := bufio.NewWriter(outputFile)
	for _, result := range results {
		_, err = writer.WriteString(gjson.MustEncodeString(result) + "\n")
		if err != nil {
			return err
		}
	}
	err = writer.Flush()
	return err
}
