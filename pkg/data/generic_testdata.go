package data

func LoadTestPrerenderData() {
	preRenderData["cluster"] = map[string]string{
		"cluster": "some-cluster",
		"region":  "us-west-2",
		"account": "1234",
		"segment": "some-segment",
	}
	preRenderData["egdata"] = map[string]string{
		"environment": "test",
	}
}
