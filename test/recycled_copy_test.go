package test

import (
	"container/list"
	"crypto/sha1"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/Bedrock-OSS/regolith/regolith"
	"github.com/otiai10/copy"
)

// assertEqualStates is use to compare two states (lists of HashPathPairs)
func assertEqualStates(stateA, stateB *list.List, t *testing.T) {
	a := stateA.Front()
	b := stateB.Front()
	for {
		if a == nil || b == nil {
			if a != b {
				t.Fatalf("A and B states have different lengths.")
			}
			break
		}
		sVal, ok := a.Value.(regolith.PathHashPair)
		if !ok {
			t.Fatalf("A element is not a PathHashPair.")
		}
		tVal, ok := b.Value.(regolith.PathHashPair)
		if !ok {
			t.Fatalf("B element is not a PathHashPair.")
		}
		if sVal.Path != tVal.Path {
			t.Fatalf(
				"A and B elements are different: %s != %s",
				sVal, tVal)
		}
		if sVal.Hash != tVal.Hash {
			t.Fatalf(
				"A and B elements are different: %s != %s",
				sVal, tVal)
		}
		// t.Log(sVal)
		a = a.Next()
		b = b.Next()
	}
}

func logState(state *list.List, t *testing.T) {
	for e := state.Front(); e != nil; e = e.Next() {
		t.Log(e.Value.(regolith.PathHashPair))
	}
}

// TestRecycledCopy tests most of the recycled_copy.go functions in one go.
func TestRecycledCopy(t *testing.T) {
	// SETUP
	regolith.InitLogging(true)
	wd, err1 := os.Getwd()
	defer os.Chdir(wd) // Go back before the test ends
	tmpDir, err2 := ioutil.TempDir("", "regolith-test")
	defer os.RemoveAll(tmpDir)
	defer os.Chdir(wd) // 'tmpDir' can't be used when we delete it
	err3 := copy.Copy( // Copy the test files
		recycledCopyData,
		tmpDir,
		copy.Options{PreserveTimes: false, Sync: false},
	)
	err4 := os.Chdir(tmpDir)
	if err := firstErr(err1, err2, err3, err4); err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}
	t.Logf("The testing directory is in: %s", tmpDir)

	// THE TEST
	// 1. Use "DeepCopyAndGetState" to copy a directory to a new location,
	// compare the state of the copied directory (returned by the copy
	// function) with the state of the original directory (from
	// "GetStateFromPath")
	source := "random_files"
	target := "random_files_1"
	t.Logf(
		"(1) Testing \"DeepCopyAndGetState\": copying \"%s\" to \"%s\"",
		source, target)
	stateTarget, err := regolith.DeepCopyAndGetState(
		source, target, sha1.New())
	if err != nil {
		t.Fatalf("Failed to copy directory: %v", err)
	}
	t.Log("Using \"GetStateFromPath\": loading the state of the source from " +
		"file structure...")
	stateSource, err := regolith.GetStateFromPath(source, sha1.New())
	if err != nil {
		t.Fatalf("Failed to get state of directory: %v", err)
	}
	t.Log("Comparing the states...")
	assertEqualStates(stateSource, stateTarget, t)
	// 2. Save the state of the copied directory and original directory
	// into a JSON file using "SavePathState" (both should be the same at this
	// point)
	t.Log("(2) Testing \"SavePathState\": saving the states into a file")
	pathStatesFilePath := "paths_states.json"
	err = regolith.SavePathState(pathStatesFilePath, source, stateSource)
	if err != nil {
		t.Fatalf("Failed to save state to a file: %v", err)
	}
	err = regolith.SavePathState(pathStatesFilePath, target, stateTarget)
	if err != nil {
		t.Fatalf("Failed to save state to a file: %v", err)
	}
	// Print the content of "path_states.json"
	// t.Log("Printing the saved state to log")
	// b, err := ioutil.ReadFile(pathStatesFilePath)
	// if err != nil {
	// 	t.Fatalf("Failed to read the file: %v", err)
	// }
	// t.Log(string(b))

	// 3. Load the state of the copied directory and original directory from
	// the JSON file using "LoadStateFromCache" and compare them to the expected
	// values.
	t.Log("(3) Testing \"LoadStateFromCache\": loading the state of the source " +
		"from a file with cached data.")
	// The directories are temporarily renamed to make sure that the state is
	// actually loaded from the file and not generated from the directory
	// structure
	source_renamed := "random_files_renamed"
	target_renamed := "random_files_1_renamed"
	err = os.Rename(source, source_renamed)
	err1 = os.Rename(target, target_renamed)
	if err := firstErr(err, err1); err != nil {
		t.Fatalf("Failed to set temporary name for directories: %v", err)
	}
	stateSourceLoaded, err := regolith.LoadStateFromCache(
		pathStatesFilePath, source)
	stateTargetLoaded, err1 := regolith.LoadStateFromCache(
		pathStatesFilePath, target)
	if err := firstErr(err, err1); err != nil {
		t.Fatalf("Failed to load state from a file: %v", err)
	}
	t.Log("Comparing loaded states with the expected values...")
	assertEqualStates(stateSourceLoaded, stateSource, t)
	assertEqualStates(stateTargetLoaded, stateTarget, t)
	// Rename the directories back
	err = os.Rename(source_renamed, source)
	err1 = os.Rename(target_renamed, target)
	if err := firstErr(err, err1); err != nil {
		t.Fatalf("Failed to restore the names of directories: %v", err)
	}
	// 4. Use "RecycledMoveOrCopy" to copy the source to the target (nothing
	// should happen because the directories are already the same)
	t.Log("(4) Testing \"RecycledMoveOrCopy\" on directories that are " +
		"already the same")
	err = regolith.RecycledMoveOrCopy(
		source, target, stateSourceLoaded, stateTargetLoaded, true)
	if err != nil {
		t.Fatalf("Failed to copy the directories: %v", err)
	}
	// assert that the stateSourceLoaded and stateTargetLoaded didn't change
	t.Log("Comparing the state values returned by \"RecycledMoveOrCopy\" " +
		"with the values before the operation (nothing should change)...")
	assertEqualStates(stateSourceLoaded, stateSource, t)
	assertEqualStates(stateTargetLoaded, stateTarget, t)
	// Reload the values from files to make sure that "RecycledMoveOrCopy"
	// returned the correct values
	stateSourceReloaded, err := regolith.GetStateFromPath(
		source, sha1.New())
	stateTargetReloaded, err1 := regolith.GetStateFromPath(
		target, sha1.New())
	if err := firstErr(err, err1); err != nil {
		t.Fatalf("Failed to load state of the path: %v", err)
	}
	t.Log("Comparing the state values returned by \"RecycledMoveOrCopy\" " +
		"with the actual values...")
	assertEqualStates(stateSourceLoaded, stateSourceReloaded, t)
	assertEqualStates(stateTargetLoaded, stateTargetReloaded, t)

	// 5. Use "RecycledMoveOrCopy" to copy the source to a new, empty location
	// All of the files should be moved to the target (the source should be
	// empty, and the target should be the same as the source)
	t.Log("(5) Testing \"RecycledMoveOrCopy\" by coping the source to empty " +
		"lcoation")
	target2 := "random_files_2"
	err = os.Mkdir(target2, 0755)
	if err != nil {
		t.Fatalf("Failed to create a directory: %v", err)
	}
	// Save the state for now
	stateSourceBefore, err := regolith.GetStateFromPath(source, sha1.New())
	// Use the "After" states in the function call (it modifies them)
	stateSourceAfter, err1 := regolith.GetStateFromPath(source, sha1.New())
	stateTarget2After, err2 := regolith.GetStateFromPath(target2, sha1.New())
	if err := firstErr(err, err1, err2); err != nil {
		t.Fatalf("Failed to get state of directory: %v", err)
	}
	err = regolith.RecycledMoveOrCopy(
		source, target2, stateSourceAfter, stateTarget2After, true)
	if err != nil {
		t.Fatalf("Failed to copy the directories: %v", err)
	}
	// Old state of source should be the same as new state of target
	t.Log("Comparing the value of the target state returned by" +
		"\"RecycledMoveOrCopy\" with the expected value...")
	assertEqualStates(stateSourceBefore, stateTarget2After, t)
	// Reload the values from files to make sure that "RecycledMoveOrCopy"
	// returned the correct values
	stateSourceReloaded, err = regolith.GetStateFromPath(source, sha1.New())
	stateTargetReloaded, err1 = regolith.GetStateFromPath(target2, sha1.New())
	if err := firstErr(err, err1); err != nil {
		t.Fatalf("Failed to load state of the path: %v", err)
	}
	t.Log("Comparing the state values returned by \"RecycledMoveOrCopy\" " +
		"with the actual values...")
	assertEqualStates(stateSourceAfter, stateSourceReloaded, t)
	assertEqualStates(stateTarget2After, stateTargetReloaded, t)

	// Move the files back to the source from target2
	err = os.RemoveAll(source)
	if err != nil {
		t.Fatalf("Failed to remove the directory: %v", err)
	}
	err = os.Rename(target2, source)
	if err != nil {
		t.Fatalf("Failed to move the files back to the source: %v", err)
	}
	// 6. Use "RecycledMoveOrCopy" to move the source to the target when the
	// target is almost exactly the same as the source (the only difference is
	// content of 1 file, 1 additional file, 1 missing file, 1 additional
	// empty directory and 1 missing empty directory).
	t.Log("(6) Testing \"RecycledMoveOrCopy\" by coping the source to " +
		"similar lcoation")
	// Modify 1 file
	modifiedFile := filepath.Join(target, "folder1/folder1.1/file1.1.1.txt")
	f, err := os.Create(modifiedFile)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	f.WriteString("This is a modified file")
	f.Close()
	// Rename 1 file (creates a new and missing file)
	missingFile := filepath.Join(target, "folder1/folder1.1/file1.1.2.txt")
	newFile := filepath.Join(target, "folder1/folder1.1/new_file.txt")
	os.Rename(missingFile, newFile)
	// Add 1 empty directory to the source
	missingEmptyDir := filepath.Join(source, "folder1/missing_empty_dir")
	err1 = os.Mkdir(missingEmptyDir, 0755)
	// Add 1 empty directory to the target
	additionalEmptyDir := filepath.Join(target, "folder1/additional_empty_dir")
	err2 = os.Mkdir(additionalEmptyDir, 0755)
	if err := firstErr(err1, err2); err != nil {
		t.Fatalf("Failed to create a directory: %v", err)
	}
	// Make sure that we have up-to-date state of source and target
	stateSourceBefore, err = regolith.GetStateFromPath(source, sha1.New())
	stateSourceAfter, err1 = regolith.GetStateFromPath(source, sha1.New())
	stateTargetAfter, err2 := regolith.GetStateFromPath(target, sha1.New())
	if err := firstErr(err, err1, err2); err != nil {
		t.Fatalf("Failed to get state of the directory: %v", err)
	}
	// Run "RecycledMoveOrCopy"
	err = regolith.RecycledMoveOrCopy(
		source, target, stateSourceAfter, stateTargetAfter, true)
	if err != nil {
		t.Fatalf("Failed to copy the directories: %v", err)
	}
	// Old state of source should be the same as new state of target
	assertEqualStates(stateSourceBefore, stateTargetAfter, t)
	// Compare the "after" states returned by the function with the actual
	// states of the source and target
	stateSource, err1 = regolith.GetStateFromPath(source, sha1.New())
	stateTarget, err2 = regolith.GetStateFromPath(target, sha1.New())
	if err := firstErr(err1, err2); err != nil {
		t.Fatalf("Failed to get state of the directory: %v", err)
	}
	t.Log("Test if \"RecycledMoveOrCopy\" returned correct source state")
	assertEqualStates(stateSourceAfter, stateSource, t)
	t.Log("Test if \"RecycledMoveOrCopy\" returned correct target state")
	assertEqualStates(stateTargetAfter, stateTarget, t)
}
