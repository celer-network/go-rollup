package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

var (
	svr     = flag.String("ethrpc", "http://127.0.0.1:8545", "ETH JSON-RPC url")
	txHash  = flag.String("tx", "", "Transaction hash")
	verbose = flag.Bool("v", false, "also print req/resp json content. default false.")
)

var hc = &http.Client{
	Timeout: time.Second * 10,
}

var cpKeys = []string{"from", "to", "gas", "gasPrice", "value"}

// Request is jsonrpc request
type Request struct {
	JSONRPC string           `json:"jsonrpc"`
	Method  string           `json:"method"`
	Params  *json.RawMessage `json:"params,omitempty"`
	ID      uint64           `json:"id"`
}

func main() {
	flag.Parse()
	reqBytes := getReqJSON("eth_getTransactionByHash", *txHash)
	resp := make(map[string]interface{})
	err := postToSvr(reqBytes, &resp)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	respErr := resp["error"]
	if respErr != nil {
		respErrMsg := respErr.(map[string]interface{})["message"].(string)
		log.Fatal().Err(errors.New(respErrMsg)).Send()
	}
	rawResult := resp["result"]
	if rawResult == nil {
		log.Fatal().Msg("No result")
	}
	result := rawResult.(map[string]interface{})
	ethCall := make(map[string]string)
	for _, k := range cpKeys {
		ethCall[k] = result[k].(string)
	}
	ethCall["data"] = result["input"].(string)
	reqBytes = getReqJSON("eth_call", ethCall, result["blockNumber"])
	resp2 := make(map[string]interface{})
	err = postToSvr(reqBytes, &resp2)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	var txt string
	if errObj, ok := resp2["error"].(map[string]interface{}); ok {
		// alchemy case, error object and error.data has raw log
		txt = strings.TrimPrefix(errObj["data"].(string), "Reverted ")
	} else {
		// infura case
		txt = resp2["result"].(string)
	}
	printReason(txt)
}

// txt is like 0x08c379a0
// 0000000000000000000000000000000000000000000000000000000000000020
// 0000000000000000000000000000000000000000000000000000000000000017
// 74686973206973206d792072657175697265206d73672e000000000000000000
// simplest parsing for now, not robust
func printReason(txt string) {
	b := hex2Bytes(txt)
	if len(b) > 68 {
		b = bytes.Trim(b, "\x00")
		log.Info().Str("reason", string(b[68:])).Send()
	} else {
		log.Info().Msg("No revert reason")
	}
}

func postToSvr(req []byte, result interface{}) error {
	if *verbose {
		log.Info().Str("request", string(req)).Send()
	}
	resp, err := hc.Post(*svr, "application/json", bytes.NewReader(req))
	if err != nil {
		return errors.New("post to svr err: " + err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("svr http err: " + resp.Status)
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.New("read resp.Body err: " + err.Error())
	}
	if *verbose {
		log.Info().Str("response", string(respBytes)).Send()
	}
	return json.Unmarshal(respBytes, result)
}

// returns marshaled bytes to post to server
func getReqJSON(method string, paramsIn ...interface{}) []byte {
	req := &Request{
		JSONRPC: "2.0",
		Method:  method,
		ID:      1,
	}
	b, _ := json.Marshal(paramsIn)
	req.Params = (*json.RawMessage)(&b)
	ret, _ := json.Marshal(req)
	return ret
}

func hex2Bytes(s string) (b []byte) {
	if len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		s = s[2:]
	}
	// hex.DecodeString expects an even-length string
	if len(s)%2 == 1 {
		s = "0" + s
	}
	b, _ = hex.DecodeString(s)
	return b
}
