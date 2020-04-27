package main

import (
	"bufio"
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

	indexFile, err := os.Open(*indexPath) // For read access.
	if err != nil {
		_, err = os.Open(*mediaPath) // For read access.
		if err != nil {
			log.Fatal(err)
			return
		}
		ffplay(strconv.Itoa(*volume), strconv.Itoa(*loop), *ss, *t, *mediaPath)
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

	ffplayShuffle := func(volume, t string, mediaPathes []string) {
		rand.Seed(time.Now().UnixNano())
		for _, index := range rand.Perm(len(mediaPathes)) {
			mediaPath := mediaPathes[index]
			fmt.Printf("%s\n", mediaPath)
			ffplay(volume, "1", "00:00", t, mediaPath)
		}
	}

	if *loop <= 0 {
		for {
			ffplayShuffle(strconv.Itoa(*volume), *t, mediaPathes)
		}
	} else {
		for ii := 0; ii < *loop; ii++ {
			ffplayShuffle(strconv.Itoa(*volume), *t, mediaPathes)
		}
	}
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

// https://www.ffmpeg.org/ffplay.html
func ffplay(volume, loop, ss, t, mediaPath string) {
	// "FFREPORT=file=ffreport.log:level=32",
	cmd := exec.Command("ffplay", "-vn", "-sn", "-nodisp", "-autoexit", "-volume", volume, "-loop", loop, "-ss", ss, mediaPath)
	if len(t) > 0 {
		cmd = exec.Command("ffplay", "-vn", "-sn", "-nodisp", "-autoexit", "-volume", volume, "-loop", loop, "-ss", ss, "-t", t, mediaPath)
	}

	stdin, _ := cmd.StdinPipe()
	defer stdin.Close()
	go func(stdin io.WriteCloser) {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Split(bufio.ScanWords)
		for scanner.Scan() {
			bb := scanner.Bytes()
			stdin.Write(bb)
			fmt.Printf("pipe %v\n", bb)
		}
	}(stdin)

	// too noisy
	// startReadingPipe(cmd)

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
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
