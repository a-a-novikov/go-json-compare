# jsoncomp
Json-compare is a simple package that allows you to easily and fastly compare two .json files. Support key and multi-key comparison.
You can also ignore certain fields' values or perform comparison insensitive to data types.

Usage
---
Compare files just as they are:

```go
import "github.com/a-a-novikov/jsoncomp"

c := jsoncomp.NewComparator(
    "expected.json",  // path of first file
    "actual.json",    // path of second file
    false,            // ignore types for same values
)

// compare "actual.json" from the perspective of "expected.json"'s structure
c.CompareWithRight()  // c.CompareWithLeft() / c.FullCompare()

// save diff logs as a text file
c.SaveDiffLogs()

// or print them into stdout
fmt.Print(c.Summary())
```
Diff-log's output example:
```text
actual.json//<array>//3//last_name
property is missing
actual.json//<array>//8//id
incorrect type: expected 4 <int>, got 4 <str> instead
actual.json//<array>
lack of items in array: expected 9 items, got only 8
actual.json//<array>//5//name
unequal values: expected Alex, got Aleks instead
-------------------
TOTAL: 4 differences
- Missing Properties: 1
- Incorrect Type: 1
- Lack of Items: 1
- Exceeding Items: 0
- Unequal Vales: 1
- Missing Items: 0
```

Set key properties or properties to ignore i.o. to perform more accurate comparisons of objects in arrays:

```go
// expected.json: {"cats": [{"id": 4, "name": "Nyan"}, {"id": 2, "name": "Marx"}, {"id": 8, "name": "Flake"}]}
// actual.json: {"cats": [{"id": 2, "name": "Marx"}, {"id": 4, "name": "Naan"}]}

c := jsoncomp.NewComparatorWithKeys(
    "expected.json",
    "actual.json",
    []string{"DATA//cats//<array>//id"},  //  <- just pass a "path" to needed property using following keywords: 
    []string{},                           //  DATA - points to the root of file 
    false,                                //  <array> - indicates array with key property's object
)

```
In this case, saved diff log would look like that:
```text
actual.json//cats//<array>
lack of items in array: expected 3 items, got only 2
actual.json//cats//<array>//0//name
unequal values: expected Nyan, got Naan instead
actual.json//cats//<array>//2
missing array item: expected <object> with id=8
```
Here's an exmaple of comparison with non-important fields set to `ignoreKeys` parameter:
```go
// expected.json: [{"id": 4, "name": "Nyan", "age": 2}, {"id": 2, "name": "Marx", "age": 7}, {"id": 8, "name": "Flake", "age": 4}]
// actual.json: [{"id": 2, "name": "Marx", "age": 7}, {"id": 4, "name": "Naan", "age": "two"}, {"id": 9, "name": "Lol", "age": 1}]

c := jsoncomp.NewComparatorWithKeys(
    "expected.json",
    "actual.json",
    []string{"DATA//<array>//id"},
    []string{"DATA//<array>//age},  // <-------
    false,
)  
```
And here the result:
```text
actual.json//<array>//0//name
unequal values: expected Nyan, got Naan instead
actual.json//<array>//2
missing array item: expected <object> with id=8
```
If you want to compare ignoring type-differences between similar values
 like `"1.4"` vs `1.4` - just set `IgnoreTypes` as true 
 param in Comparator:
```go
c := jsoncomp.NewComparatorWithKeys(
    "expected.json",
    "actual.json",
    []string{"DATA//<array>//id"},
    []string{"DATA//<array>//age},
    true,  // <-------
)  
```
