package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"
)

type Controller struct {
	cancelFuncChan chan func()
	endSignalChan  chan chan struct{}
}

func main() {
	volume := flag.Int("volume", 50, "volume")
	loop := flag.Int("loop", 1, "loop")
	ss := flag.String("ss", "00:00", "[HH:]MM:SS[.m...]")
	t := flag.String("t", "", "[HH:]MM:SS[.m...]")
	mediaPath := flag.String("media", "", "media path")
	indexPath := flag.String("index", "", "index path")
	substr := flag.String("substr", "", "substr for strings.Contains")
	extPath := flag.String("ext", "", "ext path")
	flag.Parse()

	controller := createController()
	go func(controller *Controller) {
		cancelled := make(chan struct{})
		go func(cancelled chan<- struct{}) {
			for {
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Split(bufio.ScanWords)
				for scanner.Scan() {
				}
				fmt.Println("Cancel End.")
				cancelled <- struct{}{}
			}
		}(cancelled)

		for {
			cancelProc := <-controller.cancelFuncChan
			endProc := <-controller.endSignalChan
		scan:
			for {
				select {
				case <-endProc:
					fmt.Println("Get End.")
					break scan
				case <-cancelled:
					fmt.Println("Get Cancel End.")
					cancelProc()
					break scan
				}
			}
		}
	}(controller)

	indexFile, err := os.Open(*indexPath) // For read access.
	if err != nil {
		_, err = os.Open(*mediaPath) // For read access.
		if err != nil {
			log.Fatal(err)
			return
		}
		ffplay(controller, strconv.Itoa(*volume), strconv.Itoa(*loop), *ss, *t, *mediaPath)
		return
	}

	directories := []MediaDirectory{}
	decoder := json.NewDecoder(indexFile)
	for {
		if err := decoder.Decode(&directories); err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}
	}

	exts := map[string]bool{}
	extFile, err := os.Open(*extPath) // For read access.
	if err == nil {
		scanner := bufio.NewScanner(extFile)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			ext := scanner.Text()
			if !strings.HasPrefix(ext, "#") {
				exts[ext] = true
			}
		}
	}

	mediaPathes := []string{}
	for _, directory := range directories {
		for _, mm := range directory.Images {
			_, isSupported := exts[mm.Ext]
			if len(exts) > 0 && !isSupported {
				continue
			}
			mediaPath := path.Join(mm.Directory, mm.Name)
			if len(*substr) > 0 && !strings.Contains(mediaPath, *substr) {
				continue
			}
			// fmt.Printf("contains: %s, %s\n", mediaPath, *substr)
			mediaPathes = append(mediaPathes, mediaPath)
		}
	}

	ffplayShuffle := func(controller *Controller, volume, t string, mediaPathes []string) {
		rand.Seed(time.Now().UnixNano())
		for _, index := range rand.Perm(len(mediaPathes)) {
			mediaPath := mediaPathes[index]
			fmt.Printf("%s\n", mediaPath)
			ffplay(controller, volume, "1", "00:00", t, mediaPath)
		}
	}

	if *loop <= 0 {
		for {
			ffplayShuffle(controller, strconv.Itoa(*volume), *t, mediaPathes)
		}
	} else {
		for ii := 0; ii < *loop; ii++ {
			ffplayShuffle(controller, strconv.Itoa(*volume), *t, mediaPathes)
		}
	}
}

func createController() *Controller {
	cancelFuncChan := make(chan func())
	endSignalChan := make(chan chan struct{})
	controller := &Controller{cancelFuncChan, endSignalChan}
	return controller
}

type MediaDirectory struct {
	Directory string
	Images    []Image
}

type Image struct {
	Name      string
	Height    int
	Width     int
	Ext       string
	Fid       int
	Directory string
	Size      int
}

// See https://www.ffmpeg.org/ffplay.html
func ffplay(controller *Controller, volume, loop, ss, t, mediaPath string) {
	ctx, cancel := context.WithCancel(context.Background())
	// "FFREPORT=file=ffreport.log:level=32",
	cc := exec.CommandContext(ctx, "ffplay", "-vn", "-sn", "-nodisp", "-autoexit", "-volume", volume, "-loop", loop, "-ss", ss, mediaPath)
	if len(t) > 0 {
		cc = exec.CommandContext(ctx, "ffplay", "-vn", "-sn", "-nodisp", "-autoexit", "-volume", volume, "-loop", loop, "-ss", ss, "-t", t, mediaPath)
	}

	// no reaction from ffplay
	// stdin, _ := cc.StdinPipe()
	// defer stdin.Close()

	end := make(chan struct{}, 1)

	controller.cancelFuncChan <- cancel
	controller.endSignalChan <- end

	// too noisy
	// startReadingPipe(cc)

	if err := cc.Start(); err != nil {
		log.Fatal(err)
	}
	if err := cc.Wait(); err != nil {
		log.Println(err)
	}
	fmt.Printf("End: %v\n", mediaPath)
	end <- struct{}{}
}

func startReadingPipe(cmd *exec.Cmd) {
	redirect := func(stdout io.ReadCloser, writer io.Writer) {
		scanner := bufio.NewScanner(stdout)
		scanner.Split(bufio.ScanWords)
		for scanner.Scan() {
			bb := scanner.Bytes()
			writer.Write(bb)
		}
	}

	stdout, _ := cmd.StdoutPipe()
	defer stdout.Close()
	go redirect(stdout, os.Stdout)

	stderr, _ := cmd.StderrPipe()
	defer stderr.Close()
	go redirect(stderr, os.Stderr)
}
