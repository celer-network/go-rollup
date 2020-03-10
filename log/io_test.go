package log

import (
	"bytes"
	"io"
	"os"
	"strconv"
	"sync"
	"testing"
)

func TestConcurrentWriteFile(t *testing.T) {
	t.Skip("not directly related to log module")
	epcnt := 10
	width := 100
	lineLen := 10000
	retCnt := 2
	repeatCount := 10
	sli := make([][]byte, epcnt)
	for i := 0; i < epcnt; i++ {
		line := make([]byte,width)
		f := byte(strconv.Itoa(i)[0])
		for j:=0;j<width; j++ {
			line[j] = f
		}
		line[width-1] = '\n'
		sli[i] = bytes.Repeat(line,lineLen)
		for j:=0;  j<retCnt; j++ {
			sli[i][len(sli[i])-1-j] = '\n'
		}
	}

	toWrite, err := os.OpenFile("test.log",os.O_CREATE|os.O_WRONLY,0644)
	if err != nil {
		t.Fatal("Failed to open file ",err.Error())
	}

	wg := &sync.WaitGroup{}
	wg.Add(epcnt)
	for i := 0; i < epcnt; i++ {
		go WriteIt(wg, toWrite, sli[i], repeatCount)
	}
	wg.Wait()
	toWrite.Sync()
	toWrite.Close()


	os.Remove("test2.log")
	wg.Add(epcnt)
	for i := 0; i < epcnt; i++ {
		go WriteIt2(wg, "test2.log", sli[i], repeatCount,t )
	}
	wg.Wait()
}

func WriteIt(wg *sync.WaitGroup, os io.Writer, str []byte, repeatCnt int) {
	for i:=0;i<repeatCnt;i++ {
		os.Write(str)
		sum := 0
		for j:=0 ; j<10000000 ; j++ {
			sum = j
		}
		if sum < repeatCnt {
			continue
		}
	}
	wg.Done()
}


func WriteIt2(wg *sync.WaitGroup, filename string, str []byte, repeatCnt int, t *testing.T) {
	os, err := os.OpenFile(filename,os.O_APPEND|os.O_CREATE|os.O_WRONLY,0644)
	if err != nil {
		t.Fatal("Failed to open file ",err.Error())
	}
	for i:=0;i<repeatCnt;i++ {
		os.Write(str)
	}
	os.Close()
	wg.Done()
}