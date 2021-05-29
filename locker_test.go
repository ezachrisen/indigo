package indigo_test

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/ezachrisen/indigo"
	"github.com/matryer/is"
)

func TestLocker(t *testing.T) {
	is := is.New(t)

	d := indigo.NewLocker()

	is.Equal(0, d.RuleCount())
	abc := indigo.NewRule("abc")
	abc.Rules["child1"] = indigo.NewRule("child1")
	def := indigo.NewRule("def")

	d.Add("myref", abc)
	d.Add("otherref", def)

	is.True(d.ContainsRule("abc"))
	is.True(d.ContainsRule("child1"))
	is.True(!d.ContainsRule("blah"))

	is.Equal(3, d.RuleCount())
	err := d.ReplaceRule("child1", &indigo.Rule{ID: "newchild"})
	is.NoErr(err)
	is.Equal(3, d.RuleCount())
	d.Remove("myref")
	is.True(!d.ContainsRule("abc"))

	other := d.Get("otherref")
	is.True(other != nil)

	notthere := d.Get("goofy")
	is.True(notthere == nil)

}

func TestConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	is := is.New(t)
	rand.Seed(time.Now().Unix())

	//e := indigo.NewEngine(newMockEvaluator())
	l := indigo.NewLocker()
	l.Lock()
	l.Add("ONE", makeRule())
	l.Unlock()

	var wg sync.WaitGroup

	for i := 1; i < 50_000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// Check reading known items
			l.RLock()
			is.True(l.ContainsRule("e3"))
			l.RUnlock()

			// Replace a rule
			r1 := makeTinyRule("e3")
			l.Lock()
			err := l.ReplaceRule("e3", r1)
			is.NoErr(err)
			l.Unlock()

			// Modify the containing rule
			l.Lock()
			one := l.Get("ONE")
			is.True(one != nil)
			one.Expr = "dummy"
			l.Add("ONE", one)
			l.Unlock()

			// r, err := e.Evaluate(nil, fmt.Sprintf("rule%d", i), indigo.ReturnDiagnostics(false))
			// is.NoErr(err)
			// is.Equal(r.RulesEvaluated, 10)
		}(i)
		time.Sleep(time.Duration(rand.Intn(3) * int(time.Millisecond)))
	}

	wg.Wait()
}

// func TestConcurrencyMixed(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping test in short mode.")
// 	}

// 	//dryRun := true
// 	n := 100_000
// 	randomDelay := false
// 	maxDelayMicroseconds := 100
// 	printDebug := true
// 	printDebugInterval := 10_000
// 	rand.Seed(time.Now().Unix())

// 	is := is.New(t)
// 	e := indigo.NewEngine(newMockEvaluator())

// 	// Set up a rule that we're going to evaluate over and over again
// 	r1 := makeRule()

// 	err := e.Compile(r1)
// 	is.NoErr(err)

// 	l := indigo.NewLocker()

// 	l.Add("ONE", r1)

// 	// repl := &indigo.Rule{
// 	// 	ID:   "ruleX",
// 	// 	Expr: `true`,
// 	// }

// 	// How large to make the channels depends on the capacity of the system
// 	// Because AddRule is slow, a large capacity channel will exhaust the number of
// 	// allowable goroutines.
// 	add := make(chan int, 2000)
// 	eval := make(chan int, 2000)
// 	replace := make(chan int, 2000)
// 	del := make(chan int, 2000)

// 	var adds int
// 	var evals int
// 	var replaces int
// 	var dels int

// 	//	start := time.Now()

// 	// These goroutines generate events into the "server" (the for loop at the end)
// 	// AddRule requests
// 	go func() {
// 		for i := 1; i < n; i++ {
// 			if i%printDebugInterval == 0 && printDebug {
// 				fmt.Println("---- Add ", i)
// 			}
// 			add <- i
// 			if randomDelay {
// 				time.Sleep(time.Duration(rand.Intn(maxDelayMicroseconds)) * time.Microsecond)
// 			}
// 		}
// 		close(add)
// 	}()

// 	// Evaluation requests
// 	go func() {
// 		for i := 1; i < n; i++ {
// 			if i%printDebugInterval == 0 && printDebug {
// 				fmt.Println("---- Eval ", i)
// 			}
// 			eval <- i
// 			if randomDelay {
// 				time.Sleep(time.Duration(rand.Intn(maxDelayMicroseconds)) * time.Microsecond)
// 			}

// 		}
// 		close(eval)
// 	}()

// 	// Replace rule requests
// 	go func() {
// 		for i := 1; i < n; i++ {
// 			if i%printDebugInterval == 0 && printDebug {
// 				fmt.Println("---- Replace ", i)
// 			}
// 			replace <- i
// 			if randomDelay {
// 				time.Sleep(time.Duration(rand.Intn(maxDelayMicroseconds)) * time.Microsecond)
// 			}

// 		}
// 		close(replace)
// 	}()

// 	// Delete rule requests
// 	go func() {
// 		for i := 1; i < n; i++ {
// 			if i%printDebugInterval == 0 && printDebug {
// 				fmt.Println("---- Delete ", i)
// 			}
// 			del <- i
// 			if randomDelay {
// 				time.Sleep(time.Duration(rand.Intn(maxDelayMicroseconds)) * time.Microsecond)
// 			}

// 		}
// 		close(del)
// 	}()

// 	// The "server"
// 	for eval != nil || add != nil || replace != nil || del != nil {

// 		select {

// 		case i, ok := <-add:
// 			if !ok {
// 				add = nil
// 			} else {
// 				r := makeTinyRule(fmt.Sprintf("%d", i))
// 				err := e.Compile(r)
// 				is.NoErr(err)
// 				l.Add(fmt.Sprintf("BLAH"), r)
// 				adds++
// 			}
// 			if i%printDebugInterval == 0 && printDebug {
// 				fmt.Println("---- Processed Add ", i)
// 			}

// 		case i, ok := <-eval:
// 			if !ok {
// 				eval = nil
// 			} else {
// 				r := l.Get("ONE")

// 				l.RLock()
// 				_, err := e.Eval(context.Background(), r, map[string]interface{}{})
// 				l.RUnlock()
// 				is.NoErr(err)
// 				evals++
// 				if i%printDebugInterval == 0 && printDebug {
// 					fmt.Println("---- Processed Eval ", i)
// 				}
// 			}
// 		case i, ok := <-replace:
// 			if !ok {
// 				replace = nil
// 			} else {
// 				err := l.ReplaceRule("e3", makeTinyRule("e3"))
// 				is.NoErr(err)
// 				replaces++
// 				if i%printDebugInterval == 0 && printDebug {
// 					fmt.Println("---- Processed Replace ", i)
// 				}
// 			}
// 		case i, ok := <-del:
// 			if !ok {
// 				del = nil
// 			} else {
// 				parentID := fmt.Sprintf("ruled%d", i)
// 				childID := fmt.Sprintf("rule_child_%d", i)
// 				newChild := fmt.Sprintf("rule_child_%d_new", i)

// 				r := makeParentChild(parentID, childID)
// 				rNew := makeTinyRule(newChild)
// 				// First add the rule
// 				err := e.Compile(r)
// 				is.NoErr(err)
// 				l.Add("sap", r)
// 				err = e.Compile(rNew)
// 				is.NoErr(err)
// 				err = l.ReplaceRule(childID, rNew)
// 				is.NoErr(err)
// 				dels++
// 				if i%printDebugInterval == 0 && printDebug {
// 					fmt.Println("---- Processed Delete ", i)
// 				}
// 			}
// 		}
// 	}
// 	is.True(adds == replaces && dels == replaces && replaces == evals && adds == (n-1))
// }
