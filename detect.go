package miniconda

import "github.com/paketo-buildpacks/packit"

// Detect will return a packit.DetectFunc that will be invoked during the
// detect phase of the buildpack lifecycle.
//
// Detection always passes, and will contribute a  Build Plan that provides conda.
func Detect() packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		return packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: "conda"},
				},
			},
		}, nil
	}
}
