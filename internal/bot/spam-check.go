package bot

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

type spamResult struct {
	OK bool `json:"ok"`
}

type spamMemo struct {
	OK bool
	CT int64
}

// SpamCheck send request to Combot Anti-Spam service
func (b *Bot) SpamCheck(id int) bool {
	// First check memoized value
	memoKey := "CAS" + strconv.Itoa(id)
	if ms, err := b.Memo.Get(memoKey); err == nil {
		// Try cast type
		if mr, ok := ms.(spamMemo); ok {
			// Return if value not expired (3 hours)
			if (mr.CT + 3600*3) > time.Now().Unix() {
				// Return memoized result
				return mr.OK
			}
		}
	}

	client := &http.Client{
		Timeout: 3 * time.Second, // Response timeout up to 3 seconds
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 20, // Allow 20 connections to one host
		},
	}
	request, _ := http.NewRequest(http.MethodGet, "https://api.cas.chat/check?user_id="+strconv.Itoa(id), nil)

	response, err := client.Do(request)

	if err != nil {
		b.Log.Errorf("%+v", errors.Wrap(err, "CAS: Request error"))
		return false
	}

	if response == nil {
		b.Log.Errorf("%+v", errors.New("CAS: Empty response"))
		return false
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		b.Log.Errorf("%+v", errors.Wrap(err, "CAS: Body read error"))
		return false
	}

	var spam spamResult
	err = json.Unmarshal(body, &spam)
	if err != nil {
		b.Log.Errorf("%+v", errors.Wrap(err, "CAS: JSON parsing error"))
		b.Log.Errorln("JSON parsing error body. ", string(body))
		return false
	}

	// Memoize CAS check value
	b.Memo.Set(memoKey, spamMemo{
		OK: spam.OK,
		CT: time.Now().Unix(),
	})

	return spam.OK
}
