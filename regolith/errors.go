package regolith

// Common errors and warnings used by Regolith
const (
	// Error message to display when when expecting an empty or unexisting directory
	assertEmptyOrNewDirError = "Expected a path to an empty or unexisting directory:\n%s"

	// Error message for filepath.Abs() function.
	filepathAbsError = "Failed to get absolute path to:\n%s"

	// Error message for os.Stat failore
	osStatErrorAny = "Failed to access file info for path:\n%s"

	// Error message for file or directory that doesn't exist
	osStatErrorIsNotExist = "Following path doesn't exist:\n%s"

	// Error message for os.Stat when the funciton should fail because it's
	// expected that the target path doesn't exist
	osStatExistsError = "Path already exists:\n%s"

	// Error message for handling failores of os.Rename
	osRenameError = "Failed to move file or directory:\nSource: %s\nTarget: %s"

	// Error message for handling failores of os.Copy
	osCopyError = "Failed to copy file or directory:\nSource: %s\nTarget: %s"

	// Error message displayed when mkdir (or similar function) fails
	osMkdirError = "Failed to create directory:\n%s"

	// Common Error message to be reused on top of IsDirEmpty
	isDirEmptyError = "Failed to check if path is an empty directory.\nPath: %s"

	// Error used when an empty directory is expected but it's not
	isDirEmptyNotEmptyError = "Path is an empty directory.\nPath: %s"

	// Error used when copyFileSecurityInfo fails
	copyFileSecurityInfoError = "Failed to copy ACL.\nSource: %s\nTarget: %s"

	// Error used when RevertableFsOperations.Delete fails
	revertableFsOperationsDeleteError = "Failed to delete file.\nPath: %s"

	// Error used when filepath.Rel fails
	filepathRelError = "Failed to get relative path.\nBase: %s\nTarget: %s"

	// Error used when os.Remove (or similar function) fails
	osRemoveError = "Failed to delete file or directory.\nPath: %s"

	// Error used when MoveOrCopy function fails
	moveOrCopyError = "Failed to move or copy file or directory.\nSource: %s\nTarget: %s"

	// Error used when expecting a directory but it's not
	isDirNotADirError = "Path is not a directory.\nPath: %s"

	// Error used when os.Open fails
	osOpenError = "Failed to open.\nPath: %s"

	// Error used when os.Create fails
	osCreateError = "Failed to open for writing.\nPath: %s"

	// Error used when program fails to read from opened file
	fileReadError = "Failed to read from file.\nPath: %s"

	// Error used when program fails to write to opened file
	fileWriteError = "Failed to write to file.\nPath: %s"

	// Warning used when Git is not installed
	gitNotInstalledWarning = "Git is not installed. Git is required to download " +
		"filters.\n You can download Git from https://git-scm.com/downloads"

	// Error used when certain function is not implemented on this system
	notImplementedOnThisSystemError = "Not implemented for this system."
)
