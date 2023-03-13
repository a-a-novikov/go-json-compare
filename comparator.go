package jsoncomp

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/schollz/progressbar/v3"
)

type Comparator struct {
	DiffLogger
	lName       string
	rName       string
	lData       interface{}
	rData       interface{}
	keys        []string
	ignoreKeys  []string
	IgnoreTypes bool
	ProgressBar bool
}

type jSONFile struct {
	name string
	data interface{}
}

func loadFile(name string) *jSONFile {
	fileInfo, err := os.Lstat(name)
	if err != nil { // TODO: error handling
		panic(err)
	}
	rawData, err := os.ReadFile(name)
	if err != nil { // TODO: error handling
		panic(err)
	}
	var data interface{}
	json.Unmarshal(rawData, &data)

	return &jSONFile{fileInfo.Name(), data}
}

/*
Comparator for JSON-files located at lPath and rPath.

If ignoreT set true JSONComparator will ignore type differences
between similar values like "1.4" string and 1.4 float.
*/
func NewComparator(lPath, rPath string, ignoreT bool) *Comparator {
	lFile := loadFile(lPath)
	rFile := loadFile(rPath)
	return &Comparator{
		lName:       lFile.name,
		rName:       rFile.name,
		lData:       lFile.data,
		rData:       rFile.data,
		keys:        []string{},
		ignoreKeys:  []string{},
		IgnoreTypes: ignoreT,
		ProgressBar: true,
	}
}

/*
Comparator for JSON-files located at lPath and rPath.

Uses keys and ignoreKeys to properly compare objects nested in an array.
For example, you have: {"cats": [{"id": 4, "name": "Nyan"}, {"id": 2, "name": "Marx"}]}.
If you want to set cat's "id" as a key, add "DATA.cats.<array>.id" to keys.
DATA points out to the root of your JSON and <array> indicates object with key is nested in an array.
ignoreKeys are used to ignore mismatches for particular keys.
If ignoreT set true JSONComparator will ignore type differences
between similar values like "1.4" string and 1.4 float.
*/
func NewComparatorWithKeys(lPath, rPath string, keys, ignoreKeys []string, ignoreT bool) *Comparator {
	lFile := loadFile(lPath)
	rFile := loadFile(rPath)
	return &Comparator{
		lName:       lFile.name,
		rName:       rFile.name,
		lData:       lFile.data,
		rData:       rFile.data,
		keys:        keys,
		ignoreKeys:  ignoreKeys,
		IgnoreTypes: ignoreT,
		ProgressBar: true,
	}
}

// Looks for how the right JSON-file differs from the right one.
func (c *Comparator) CompWithRight() {
	c.clearTemp()
	c.setKeyRoot("DATA", c.rName)
	c.compare(false)
	c.setKeyRoot(c.rName, "DATA")
	c.setupSummary()
}

// Looks for how the left JSON-file differs from the left one.
func (c *Comparator) CompWithLeft() {
	c.clearTemp()
	c.setKeyRoot("DATA", c.lName)
	c.compare(true)
	c.setKeyRoot(c.lName, "DATA")
	c.setupSummary()
}

// Looks for differences between files from both perspectives.
func (c *Comparator) CompareFull() {
	c.clearTemp()
	c.setKeyRoot("DATA", c.lName)
	c.compare(false)
	c.setKeyRoot(c.rName, c.rName)
	c.compare(true)
	c.setKeyRoot(c.lName, "DATA")
	c.setupSummary()
}

func (c *Comparator) compare(withLeft bool) {
	_, leftIsArr := c.lData.([]interface{})
	_, rightIsArr := c.rData.([]interface{})
	root, lData, rData := c.lName, c.lData, c.rData
	if withLeft {
		root, lData, rData = c.rName, c.rData, c.lData
	}
	if leftIsArr && rightIsArr {
		c.compareArr(root, lData, rData)
	} else if !leftIsArr && !rightIsArr {
		c.compareObj(root, lData, rData)
	} else {
		c.newIncorrType(lData, rData)
	}
}

func (c *Comparator) compareObj(p string, exp, act interface{}) {
	parsedExp := exp.(map[string]interface{})
	parsedAct := act.(map[string]interface{})
	for k, v := range parsedExp {
		c.setCurrPath(p, k)
		if _, inActData := parsedAct[k]; !inActData {
			c.newMissProp()
			continue
		}
		if _, isObj := v.(map[string]interface{}); isObj {
			c.compareObj(c.currPath, v, parsedAct[k])
		} else if _, isArr := v.([]interface{}); isArr {
			c.compareArr(c.currPath, v, parsedAct[k])
		} else if v != parsedAct[k] && !c.ignoreCurrKey() {
			c.compareValues(v, parsedAct[k])
		}
	}
}

func (c *Comparator) compareArr(p string, exp, act interface{}) {
	c.setCurrPath(p, "<array>")
	keys := c.matchingKeys()
	if len(keys) > 0 {
		c.compareArrByKey(c.currPath, exp, act, keys)
	} else {
		c.compareArrByOrder(c.currPath, exp, act)
	}
}

func (c *Comparator) compareArrByOrder(p string, exp, act interface{}) {
	parsedExp := exp.([]interface{})
	parsedAct := act.([]interface{})
	expLen, actLen := len(parsedExp), len(parsedAct)
	c.compareArrLen(p, expLen, actLen)
	var bar *progressbar.ProgressBar
	if c.ProgressBar {
		bar = progressbar.Default(int64(expLen))
	}
	for i, v := range parsedExp {
		if c.ProgressBar {
			bar.Add(1)
		}
		c.setCurrPath(p, strconv.Itoa(i))
		if i >= actLen {
			break
		}
		if _, isObj := v.(map[string]interface{}); isObj {
			c.compareObj(c.currPath, v, parsedAct[i])
			continue
		}
		if _, isArr := v.([]interface{}); isArr {
			c.compareArr(c.currPath, v, parsedAct[i])
			continue
		}
		if v != parsedAct[i] {
			c.compareValues(v, parsedAct[i])
		}
	}
}

func (c *Comparator) compareArrByKey(p string, exp, act interface{}, keys []string) {
	parsedExp := exp.([]interface{})
	parsedAct := act.([]interface{})
	expLen, actLen := len(parsedExp), len(parsedAct)
	c.compareArrLen(p, expLen, actLen)
	var bar *progressbar.ProgressBar
	if c.ProgressBar {
		bar = progressbar.Default(int64(expLen))
	}
	for i, v := range parsedExp {
		if c.ProgressBar {
			bar.Add(1)
		}
		c.setCurrPath(p, strconv.Itoa(i))
		if _, isObj := v.(map[string]interface{}); isObj {
			targetKeyVals := c.TargetKeyValMap(v, keys)
			relevantActIdx, ok := c.relevantKeyValIdx(parsedAct, targetKeyVals)
			if !ok {
				c.newMissItem(v, targetKeyVals)
				continue
			}
			c.compareObj(c.currPath, v, parsedAct[relevantActIdx])
			parsedAct = append(parsedAct[:relevantActIdx], parsedAct[relevantActIdx+1:]...)
			continue
		}
		if _, isArr := v.([]interface{}); isArr {
			c.compareArr(c.currPath, v, parsedAct[i])
			parsedAct = append(parsedAct[:i], parsedAct[i+1:]...)
			continue
		}
		if v != parsedAct[i] {
			c.compareValues(v, parsedAct[i])
		}
	}
}

func (c *Comparator) clearTemp() {
	c.Log = []string{}
	c.currPath = ""
}

func (c *Comparator) setKeyRoot(old, new string) {
	var keys []string
	for _, key := range c.keys {
		keys = append(keys, strings.Replace(key, old, new, -1))
	}
	c.keys = keys
}

func (c *Comparator) ignoreCurrKey() bool {
	re := regexp.MustCompile(`\[[^()]*\]`)
	p := re.ReplaceAllString(c.currPath, "")
	p = strings.Replace(p, c.rName, "DATA", 1)
	p = strings.Replace(p, c.rName, "DATA", 1)
	p = strings.Replace(p, "////", "//", -1)
	for _, keyToIgnore := range c.ignoreKeys {
		if p == keyToIgnore {
			return true
		}
	}
	return false
}

func (c *Comparator) compareValues(exp, act interface{}) {
	if exp == act {
		return
	}
	if !c.IgnoreTypes {
		c.newUnequalVal(exp, act)
		return
	}
	expInt, expIsInt := exp.(int)
	actInt, actIsInt := act.(int)
	if expIsInt && actIsInt && expInt == actInt {
		return
	}
	expFl, expIsFl := exp.(float64)
	actFl, actIsFl := act.(float64)
	if expIsFl && actIsFl && expFl == actFl {
		return
	}
	expStr, expIsStr := exp.(string)
	actStr, actIsStr := act.(string)
	if expIsStr && actIsStr && expStr == actStr {
		return
	}
	expBool, expIsBool := exp.(bool)
	actBool, actIsBool := act.(bool)
	if expIsBool && actIsBool && expBool == actBool {
		return
	}
	switch exp.(type) {
	case string:
		if reflect.ValueOf(exp).String() == fmt.Sprint(act) {
			return
		}
	case int:
		actInt, err := strconv.Atoi(fmt.Sprint(act))
		if err == nil && exp == actInt {
			return
		}
	case float64:
		actFl, err := strconv.ParseFloat(fmt.Sprint(act), 64)
		if err == nil && exp == actFl {
			return
		}
	case bool:
		if (exp == true && fmt.Sprint(act) == "true") || (exp == false && fmt.Sprint(act) == "false") {
			return
		}
	case nil:
		if fmt.Sprint(act) == "null" || fmt.Sprint(act) == "nil" {
			return
		}
	}
	c.newUnequalVal(exp, act)
}

func (c *Comparator) matchingKeys() []string {
	if len(c.keys) == 0 {
		return []string{}
	}
	var mKeys []string
	for _, k := range c.keys {
		fmt.Print(k)
		cleanK := strings.Replace(k, c.currPath+"//", "", 1)
		if !strings.Contains(cleanK, "//") {
			mKeys = append(mKeys, cleanK)
		}
	}
	return mKeys
}

func (c *Comparator) compareArrLen(p string, expLen, actLen int) {
	if expLen > actLen {
		c.newLackItems(expLen, actLen)
	} else if expLen < actLen {
		c.newExceedItems(expLen, actLen)
	}
}

func (c *Comparator) TargetKeyValMap(src interface{}, keys []string) map[string]interface{} {
	parsedSrc := src.(map[string]interface{})
	res := make(map[string]interface{})
	for _, k := range keys {
		res[k] = parsedSrc[k]
	}
	return res
}

func (c *Comparator) relevantKeyValIdx(src []interface{}, targetKeyVal map[string]interface{}) (idx int, ok bool) {
	for i, v := range src {
		parsedV := v.(map[string]interface{})
		isTarget := true
		for tKey, tVal := range targetKeyVal {
			if parsedV[tKey] != tVal {
				isTarget = false
			}
		}
		if isTarget {
			return i, true
		}
	}
	return 0, false
}
