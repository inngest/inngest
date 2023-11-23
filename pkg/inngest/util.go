package inngest

func GetFailureHandlerSlug(functionSlug string) string {
	return functionSlug + "-failure"
}
