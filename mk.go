package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

// True if messages should be printed without fancy colors.
var nocolor bool = false

// True if we are no actualyl executing any recipes or updating any timestamps.
var dryrun bool = false

// The maximum number of times an rule may be applied.
const max_rule_cnt = 3

// Ansi color codes.
const (
	colorDefault string = "\033[0m"
	colorBlack   string = "\033[30m"
	colorRed     string = "\033[31m"
	colorGreen   string = "\033[32m"
	colorYellow  string = "\033[33m"
	colorBlue    string = "\033[34m"
	colorMagenta string = "\033[35m"
)

func mk(rs *ruleSet, target string, dryrun bool) {
	g := buildgraph(rs, target)
	//g.visualize(os.Stdout)
	mkNode(g, g.root)
}

// Build a target in the graph.
//
// This selects an appropriate rule (edge) and builds all prerequisites
// concurrently.
//
// TODO: control the number of concurrent rules being built. This would involve
// some sort of global counter protected by a mutex, so we wait our turn to
// execute our rule.
//
func mkNode(g *graph, u *node) {
	// try to claim on this node
	u.mutex.Lock()
	if u.status != nodeStatusReady {
		u.mutex.Unlock()
		return
	} else {
		u.status = nodeStatusStarted
	}
	u.mutex.Unlock()

	// when finished, notify the listeners
	finalstatus := nodeStatusDone
	defer func() {
		u.mutex.Lock()
		u.status = finalstatus
		u.mutex.Unlock()
		for i := range u.listeners {
			u.listeners[i] <- u.status
		}
	}()

	// there's no fucking rules, dude
	if len(u.prereqs) == 0 {
		if !u.r.attributes.virtual && !u.exists {
			wd, _ := os.Getwd()
			mkError(fmt.Sprintf("don't know how to make %s in %s", u.name, wd))
		}
		return
	}

	// there should otherwise be exactly one edge with an associated rule
	prereqs := make([]*node, 0)
	var e *edge = nil
	for i := range u.prereqs {
		if u.prereqs[i].r != nil {
			e = u.prereqs[i]
		}
		if u.prereqs[i].v != nil {
			prereqs = append(prereqs, u.prereqs[i].v)
		}
	}

	// this should have been caught during graph building
	if e == nil {
		wd, _ := os.Getwd()
		mkError(fmt.Sprintf("don't know how to make %s in %s", u.name, wd))
	}

	prereqstat := make(chan nodeStatus)
	pending := 0

	// build prereqs that need building
	e.r.mutex.Lock()
	for i := range prereqs {
		prereqs[i].mutex.Lock()
		// needs to be built?
		if !prereqs[i].exists || e.r.attributes.virtual || (u.exists && u.t.Before(prereqs[i].t)) {
			switch prereqs[i].status {
			case nodeStatusReady:
				go mkNode(g, prereqs[i])
				fallthrough
			case nodeStatusStarted:
				prereqs[i].listeners = append(prereqs[i].listeners, prereqstat)
				pending++
			}
		}
		prereqs[i].mutex.Unlock()
	}
	e.r.mutex.Unlock()

	// wait until all the prereqs are built
	//fmt.Printf("%s: %d\n", u.name, pending)
	for pending > 0 {
		//for i := range prereqs {
		//fmt.Println(prereqs[i].name)
		//}

		s := <-prereqstat
		pending--
		if s == nodeStatusFailed {
			finalstatus = nodeStatusFailed
		}
	}

	// execute the recipe, unless the prereqs failed
	if finalstatus != nodeStatusFailed {
		mkPrintMessage("mking " + u.name)
		if !dorecipe(u.name, u, e) {
			finalstatus = nodeStatusFailed
		}
	}

	mkPrintSuccess("finished mking " + u.name)
}

func mkError(msg string) {
	if !nocolor {
		os.Stderr.WriteString(colorRed)
	}
	fmt.Fprintf(os.Stderr, "mk: %s\n", msg)
	if !nocolor {
		os.Stderr.WriteString(colorDefault)
	}
	os.Exit(1)
}

func mkPrintSuccess(msg string) {
	if nocolor {
		fmt.Println(msg)
	} else {
		fmt.Printf("%s%s%s\n", colorGreen, msg, colorDefault)
	}
}

func mkPrintMessage(msg string) {
	if nocolor {
		fmt.Println(msg)
	} else {
		fmt.Printf("%s%s%s\n", colorBlue, msg, colorDefault)
	}
}

func mkPrintRecipe(msg string) {
	if !nocolor {
		os.Stdout.WriteString(colorYellow)
	}
	printIndented(os.Stdout, msg)
	if !nocolor {
		os.Stdout.WriteString(colorDefault)
	}
}

func main() {
	var mkfilepath string
	var dryrun bool
	flag.StringVar(&mkfilepath, "f", "mkfile", "use the given file as mkfile")
	flag.BoolVar(&dryrun, "n", false, "print commands without actually executing")
	flag.Parse()

	mkfile, err := os.Open(mkfilepath)
	if err != nil {
		mkError("no mkfile found")
	}
	input, _ := ioutil.ReadAll(mkfile)
	mkfile.Close()

	rs := parse(string(input), mkfilepath)
	targets := flag.Args()

	// build the first non-meta rule in the makefile, if none are given explicitly
	for i := range rs.rules {
		if !rs.rules[i].ismeta {
			for j := range rs.rules[i].targets {
				targets = append(targets, rs.rules[i].targets[j].spat)
			}
			break
		}
	}

	if len(targets) == 0 {
		fmt.Println("mk: nothing to mk")
		return
	}

	// TODO: For multiple targets, we should add a dummy rule that depends on
	// all let mk handle executing each.
	for _, target := range targets {
		mk(rs, target, dryrun)
	}
}
