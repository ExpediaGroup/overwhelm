package reference

func LoadTestPrerenderData() {
	preRenderData["cluster"] = map[string]string{
		"cluster": "rcp-xyz",
		"region":  "us-west-2",
		"account": "1234",
		"segment": "oos",
	}
	preRenderData["egdata"] = map[string]string{
		"environment": "test",
	}
}
