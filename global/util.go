package global

import "log"

func InfoLog(module string, detail string, params interface{}) {
	log.Printf("%s %s %v", module, detail, params)
}
