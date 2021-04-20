package miniconda

const (
	// This is the key name that we use to store the sha of the script we
	// download in the layer metadata, which is used to determine if the conda
	// layer can be resued on during a rebuild.
	DepKey = "dependency-sha"
)
