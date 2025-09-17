package kv_server_lock_mechanism_unstable_network

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/anishathalye/porcupine"
)

const (
	TAG_CHECKER   string = "$ Checker"
	TAG_PARTITION string = "$ Failure"
	TAG_INFO      string = "$ Test Info"
)

const (
	COLOR_INFO    string = "#FAFAFA"
	COLOR_NEUTRAL string = "#FFECB3"
	COLOR_SUCCESS string = "#C8E6C9"
	COLOR_FAILURE string = "#FFCDD2"
	COLOR_FAULT   string = "#B3E5FC"
	COLOR_USER    string = "#FFF176"
)

// Using global variable feels disturbing, but also can't figure out a better
// way to support user-level annotations. An alternative would be passing an
// Annotation object to the start-up function of servers and clients, but that
// doesn't feel better.
//
// One potential problem with using a global Annotation object is that when
// running multiple test cases, some zombie threads in previous test cases could
// interfere the current one. An ad-hoc fix at the user level would be adding
// annotations only if the killed flag on the server is not set.
var annotation *Annotation = mkAnnotation()

func FinalizeAnnotations(end string) []porcupine.Annotation {
	annotations := annotation.finalize()

	// XXX: Make the last annotation an interval one to work around Porcupine's
	// issue. Consider removing this once the issue is fixed.
	t := timestamp()
	aend := porcupine.Annotation{
		Tag:             TAG_INFO,
		Start:           t,
		End:             t + 1000,
		Description:     end,
		Details:         end,
		BackgroundColor: COLOR_INFO,
	}
	annotations = append(annotations, aend)

	return annotations
}

func GetAnnotationFinalized() bool {
	return annotation.isFinalized()
}

func (an *Annotation) finalize() []porcupine.Annotation {
	an.mu.Lock()
	defer an.mu.Unlock()

	x := an.annotations

	t := timestamp()
	for tag, cont := range an.continuous {
		a := porcupine.Annotation{
			Tag:             tag,
			Start:           cont.start,
			End:             t,
			Description:     cont.desp,
			Details:         cont.details,
			BackgroundColor: cont.bgcolor,
		}
		x = append(x, a)
	}

	an.finalized = true
	return x
}

func mkAnnotation() *Annotation {
	an := Annotation{
		mu:          new(sync.Mutex),
		annotations: make([]porcupine.Annotation, 0),
		continuous:  make(map[string]Continuous),
		finalized:   false,
	}

	return &an
}

///
/// Public interface.
///

type Annotation struct {
	mu          *sync.Mutex
	annotations []porcupine.Annotation
	continuous  map[string]Continuous
	finalized   bool
}

///
/// Internal.
///

func timestamp() int64 {
	return int64(time.Since(time.Unix(0, 0)))
}

func (an *Annotation) isFinalized() bool {
	annotation.mu.Lock()
	defer annotation.mu.Unlock()

	return annotation.finalized
}

type Continuous struct {
	start   int64
	desp    string
	details string
	bgcolor string
}

func (an *Annotation) cleanup(failed bool, end string) {
	enabled := os.Getenv("VIS_ENABLE")
	if enabled == "never" || (!failed && enabled != "always") || an.isFinalized() {
		// Simply clean up the annotations without producing the vis file if
		// VIS_ENABLE is set to "never", OR if the test passes AND VIS_ENABLE is
		// not set to "always", OR the current test has already been finalized
		// (because CheckPorcupine has already produced a vis file).
		an.clear()
		return
	}

	annotations := an.finalize()
	if len(annotations) == 0 {
		// Skip empty annotations.
		return
	}

	// XXX: Make the last annotation an interval one to work around Porcupine's
	// issue. Consider removing this once the issue is fixed.
	t := timestamp()
	aend := porcupine.Annotation{
		Tag:             TAG_INFO,
		Start:           t,
		End:             t + 1000,
		Description:     end,
		Details:         end,
		BackgroundColor: COLOR_INFO,
	}
	annotations = append(annotations, aend)

	fpath := os.Getenv("VIS_FILE")
	var file *os.File
	var err error
	if fpath == "" {
		// Save the vis file in a temporary file.
		file, err = os.CreateTemp("", "porcupine-*.html")
	} else {
		file, err = os.OpenFile(fpath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	}
	if err != nil {
		fmt.Printf("info: failed to open visualization file %s (%v)\n", fpath, err)
		return
	}

	// Create a fresh linearization info without any client operations and use
	// models.KvModel simply as a placeholder.
	info := porcupine.LinearizationInfo{}
	info.AddAnnotations(annotations)
	porcupine.Visualize(KvModel, info, file)
	fmt.Printf("info: wrote visualization to %s\n", file.Name())
}

func (an *Annotation) clear() {
	an.mu.Lock()
	an.annotations = make([]porcupine.Annotation, 0)
	an.continuous = make(map[string]Continuous)
	an.finalized = false
	an.mu.Unlock()
}
