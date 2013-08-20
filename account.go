package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

func NewAccount(clientId, apiKey string) *Account {
	return &Account{ApiKey: apiKey, ClientId: clientId}
}

type Account struct {
	Name     string
	ApiKey   string
	SshKey   int
	ClientId string

	RegionId int
	SizeId   int
	ImageId  int

	cachedSizes   map[int]string
	cachedImages  map[int]string
	cachedRegions map[int]string
}

type SshKeysResponse struct {
	Status string    `json:"status"`
	Keys   []*SshKey `json:"ssh_keys"`
}

func (account *Account) SshKeys() (keys []*SshKey, e error) {
	rsp := &SshKeysResponse{}
	e = account.loadResource("/ssh_keys", rsp)
	if e != nil {
		return keys, e
	}
	return rsp.Keys, nil
}

func (account *Account) Sizes() (sizes []*Size, e error) {
	rsp := &SizeResponse{}
	e = account.loadResource("/sizes", rsp)
	if e != nil {
		return sizes, e
	}
	return rsp.Sizes, nil
}

func (account *Account) CachedImages() (hash map[int]string, e error) {
	if account.cachedImages == nil {
		account.cachedImages = make(map[int]string)
		images, e := account.Images()
		if e != nil {
			return hash, e
		}
		for _, image := range images {
			account.cachedImages[image.Id] = image.Name
		}
	}
	return account.cachedImages, e
}

func (a *Account) RebuildDroplet(id int, imageId int) (*EventResponse, error) {
	logger.Infof("rebuilding droplet %d with image %d", id, imageId)
	rsp := &EventResponse{}
	path := fmt.Sprintf("/droplets/%d/rebuild?image_id=%d", id, imageId)
	if e := a.loadResource(path, rsp); e != nil {
		return nil, e
	}
	if rsp.Status != "OK" {
		return rsp, fmt.Errorf("error rebuilding droplet")
	}
	return rsp, nil
}

func (account *Account) DestroyDroplet(id int) (*EventResponse, error) {
	rsp := &EventResponse{}
	if e := account.loadResource(fmt.Sprintf("/droplets/%d/destroy", id), rsp); e != nil {
		return nil, e
	}
	if rsp.Status != "OK" {
		return rsp, fmt.Errorf("error destroying droplet: %s", rsp.ErrorMessage)
	}
	return rsp, nil
}

func (account *Account) ImageName(i int) string {
	if images, e := account.CachedImages(); e != nil {
		return ""
	} else {
		return images[i]
	}
}

func (account *Account) CachedRegions() (hash map[int]string, e error) {
	if account.cachedRegions == nil {
		account.cachedRegions = make(map[int]string)
		regions, e := account.Regions()
		if e != nil {
			return hash, e
		}
		for _, region := range regions {
			account.cachedRegions[region.Id] = region.Name
		}
	}
	return account.cachedRegions, e
}

func (account *Account) RegionName(i int) string {
	if regions, e := account.CachedRegions(); e != nil {
		return ""
	} else {
		return regions[i]
	}
}

func (account *Account) CachedSizes() (hash map[int]string, e error) {
	if account.cachedSizes == nil {
		account.cachedSizes = make(map[int]string)
		sizes, e := account.Sizes()
		if e != nil {
			return hash, e
		}
		for _, size := range sizes {
			account.cachedSizes[size.Id] = size.Name
		}
	}
	return account.cachedSizes, e
}

func (account *Account) SizeName(i int) string {
	if sizes, e := account.CachedSizes(); e != nil {
		return ""
	} else {
		return sizes[i]
	}
}

func (account *Account) DefaultDroplet() (droplet *Droplet) {
	droplet = &Droplet{}
	if account.RegionId > 0 {
		droplet.RegionId = account.RegionId
	}
	if account.SizeId > 0 {
		droplet.SizeId = account.SizeId
	}
	if account.ImageId > 0 {
		droplet.ImageId = account.ImageId
	}
	return droplet
}

type ErrorResponse struct {
	Status string `json:"status"`
	Error  string `json:"error_message"`
}

func (account *Account) getJSON(path string) (b []byte, e error) {
	url := fmt.Sprintf("%s%s", API_ROOT, path)
	if !strings.Contains(url, "?") {
		url += "?"
	} else {
		url += "&"
	}
	url += fmt.Sprintf("client_id=%s&api_key=%s", account.ClientId, account.ApiKey)
	logger.Debug("fetching", url)
	started := time.Now()
	rsp, e := http.Get(url)
	if e != nil {
		return
	}
	logger.Debugf("got status %s", rsp.Status)
	if !strings.HasPrefix(rsp.Status, "2") {
		b, e := ioutil.ReadAll(rsp.Body)
		errorRsp := &ErrorResponse{}
		e = json.Unmarshal(b, errorRsp)
		if e == nil {
			return b, fmt.Errorf("got status %s and error %q when fetching %s", rsp.Status, errorRsp.Error, url)
		}
		return b, fmt.Errorf("got status %s when fetching %s", rsp.Status, url)
	}
	logger.Debugf("fetched %s in %.06f", url, time.Now().Sub(started).Seconds())
	b, e = ioutil.ReadAll(rsp.Body)
	return
}

func (account *Account) loadResource(path string, i interface{}) (e error) {
	body, e := account.getJSON(path)
	if e != nil {
		logger.Error(string(body))
		return
	}
	e = json.Unmarshal(body, i)
	if e != nil {
		logger.Error(string(body))
	}
	return e
}

func (self *Account) Droplets() (droplets []*Droplet, e error) {
	dropletResponse := &DropletsResponse{}
	e = self.loadResource("/droplets", dropletResponse)
	if e != nil {
		return droplets, e
	}
	droplets = dropletResponse.Droplets
	for i := range droplets {
		droplets[i].Account = self
	}
	return droplets, nil
}
