package jsoncomp

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Logger interface {
	GetSummary() string
}

type DiffLogger struct {
	Log         []string
	currPath    string
	missingProp int
	incorrType  int
	lackOfItems int
	exceedItems int
	uneqVal     int
	missItems   int
}

// Returns formatted summary for last performed comparison.
func (l *DiffLogger) Summary() string {
	return strings.Join(l.Log, "\n")
}

// Saves logs for last performed comparison as a text file.
func (c *Comparator) SaveDiffLogs() {
	curTime := time.Now().Format("2006_01_02_15_04_05")
	fName := "json_comp_" + curTime
	f, err := os.Create(fName)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	for _, log := range c.Log {
		f.Write([]byte(log))
	}
}

func (l *DiffLogger) setupSummary() {
	summary := fmt.Sprintf(
		`-------------------
TOTAL: %d differences
- Missing Properties: %d
- Incorrect Type: %d
- Lack of Items: %d
- Exceeding Items: %d
- Unequal Vales: %d
- Missing Items: %d
`,
		len(l.Log),
		l.missingProp,
		l.incorrType,
		l.lackOfItems,
		l.exceedItems,
		l.uneqVal,
		l.missItems,
	)
	l.Log = append(l.Log, summary)
}

func (l *DiffLogger) newIncorrType(exp, act interface{}) {
	l.incorrType++
	expT := l.toJSONType(exp)
	actT := l.toJSONType(act)
	msg := l.currPath + fmt.Sprintf(
		"\nincorrect type: expected %v %s, got %v %s instead\n",
		exp, expT, act, actT,
	)
	l.Log = append(l.Log, msg)
}

func (l *DiffLogger) newMissProp() {
	l.missingProp++
	msg := l.currPath + "\nproperty is missing"
	l.Log = append(l.Log, msg)
}

func (l *DiffLogger) newUnequalVal(exp, act interface{}) {
	l.uneqVal++
	if fmt.Sprintf("%T", exp) != fmt.Sprintf("%T", act) {
		l.newIncorrType(exp, act)
		return
	}
	msg := l.currPath + fmt.Sprintf(
		"\nunequal values: expected %v, got %v instead",
		exp, act,
	)
	l.Log = append(l.Log, msg)
}

func (l *DiffLogger) newLackItems(expLen, actLen int) {
	l.lackOfItems++
	msg := l.currPath + fmt.Sprintf(
		"\nlack of items in array: expected %d items, got only %d",
		expLen, actLen,
	)
	l.Log = append(l.Log, msg)
}

func (l *DiffLogger) newExceedItems(expLen, actLen int) {
	l.exceedItems++
	msg := l.currPath + fmt.Sprintf(
		"\ntoo much items in array: expected %d items, got %d",
		expLen, actLen,
	)
	l.Log = append(l.Log, msg)
}

func (l *DiffLogger) newMissItem(exp interface{}, tKeyVals map[string]interface{}) {
	l.missItems++
	var parsedTarget []string
	for k, v := range tKeyVals {
		parsedTarget = append(parsedTarget, fmt.Sprintf("%v: %v", k, v))
	}
	msg := l.currPath + "\nmissing array item: expected <object> with " + strings.Join(parsedTarget, ", ")
	l.Log = append(l.Log, msg)
}

func (l *DiffLogger) toJSONType(v interface{}) string {
	switch v.(type) {
	case int:
		return "<int>"
	case float64:
		return "<float>"
	case string:
		return "<str>"
	case bool:
		return "<bool>"
	case map[string]interface{}:
		return "<object>"
	case []interface{}:
		return "<array>"
	default:
		return "<null>"
	}
}

func (l *DiffLogger) setCurrPath(prevP, key string) {
	if prevP != "" {
		l.currPath = prevP + "//" + key
	} else {
		l.currPath = "//" + key
	}
}
