package jest

import (
	sitter "github.com/smacker/go-tree-sitter"

	"github.com/specvital/core/domain"
	"github.com/specvital/core/parser"
)

func addTestToTarget(test domain.Test, parentSuite *domain.TestSuite, file *domain.TestFile) {
	if parentSuite != nil {
		parentSuite.Tests = append(parentSuite.Tests, test)
	} else {
		file.Tests = append(file.Tests, test)
	}
}

func addSuiteToTarget(suite domain.TestSuite, parentSuite *domain.TestSuite, file *domain.TestFile) {
	if parentSuite != nil {
		parentSuite.Suites = append(parentSuite.Suites, suite)
	} else {
		file.Suites = append(file.Suites, suite)
	}
}

func parseCallbackBody(callback *sitter.Node, source []byte, filename string, file *domain.TestFile, suite *domain.TestSuite) {
	body := callback.ChildByFieldName("body")
	if body != nil {
		parseJestNode(body, source, filename, file, suite)
	}
}

func resolveEachNames(template string, testCases []string) []string {
	if len(testCases) == 0 {
		return []string{template + dynamicCasesSuffix}
	}

	names := make([]string, len(testCases))
	for i, testCase := range testCases {
		names[i] = formatEachName(template, testCase)
	}

	return names
}

func processEachTests(callNode *sitter.Node, testCases []string, nameTemplate string, filename string, file *domain.TestFile, parentSuite *domain.TestSuite, status domain.TestStatus) {
	names := resolveEachNames(nameTemplate, testCases)

	for _, name := range names {
		test := domain.Test{
			Name:     name,
			Status:   status,
			Location: parser.GetLocation(callNode, filename),
		}

		addTestToTarget(test, parentSuite, file)
	}
}

func processEachSuites(callNode *sitter.Node, testCases []string, nameTemplate string, callback *sitter.Node, source []byte, filename string, file *domain.TestFile, parentSuite *domain.TestSuite, status domain.TestStatus) {
	if callback == nil {
		return
	}

	names := resolveEachNames(nameTemplate, testCases)

	for _, name := range names {
		suite := domain.TestSuite{
			Name:     name,
			Status:   status,
			Location: parser.GetLocation(callNode, filename),
		}

		parseCallbackBody(callback, source, filename, file, &suite)
		addSuiteToTarget(suite, parentSuite, file)
	}
}

func processEachCall(outerCall, innerCall, outerArgs *sitter.Node, source []byte, filename string, file *domain.TestFile, currentSuite *domain.TestSuite) {
	innerFunc := innerCall.ChildByFieldName("function")
	innerArgs := innerCall.ChildByFieldName("arguments")

	if innerFunc == nil || innerArgs == nil {
		return
	}

	funcName, status := parseFunctionName(innerFunc, source)
	if funcName == "" {
		return
	}

	testCases := extractEachTestCases(innerArgs, source)
	nameTemplate := extractTestName(outerArgs, source)
	callback := findCallback(outerArgs)

	switch funcName {
	case funcDescribe + "." + modifierEach:
		processEachSuites(outerCall, testCases, nameTemplate, callback, source, filename, file, currentSuite, status)
	case funcIt + "." + modifierEach, funcTest + "." + modifierEach:
		processEachTests(outerCall, testCases, nameTemplate, filename, file, currentSuite, status)
	}
}

func processTest(callNode *sitter.Node, args *sitter.Node, source []byte, filename string, file *domain.TestFile, parentSuite *domain.TestSuite, status domain.TestStatus) {
	name := extractTestName(args, source)
	if name == "" {
		return
	}

	test := domain.Test{
		Name:     name,
		Status:   status,
		Location: parser.GetLocation(callNode, filename),
	}

	addTestToTarget(test, parentSuite, file)
}

func processSuite(callNode *sitter.Node, args *sitter.Node, source []byte, filename string, file *domain.TestFile, parentSuite *domain.TestSuite, status domain.TestStatus) {
	name := extractTestName(args, source)
	if name == "" {
		return
	}

	callback := findCallback(args)
	if callback == nil {
		return
	}

	suite := domain.TestSuite{
		Name:     name,
		Status:   status,
		Location: parser.GetLocation(callNode, filename),
	}

	parseCallbackBody(callback, source, filename, file, &suite)
	addSuiteToTarget(suite, parentSuite, file)
}
