package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TestReport 测试报告结构
type TestReport struct {
	Timestamp      string      `json:"timestamp"`
	GitCommit      string      `json:"git_commit"`
	GitBranch      string      `json:"git_branch"`
	GoVersion      string      `json:"go_version"`
	Summary        TestSummary `json:"summary"`
	TestSuites     []TestSuite `json:"test_suites"`
	BuildArtifacts []BuildInfo `json:"build_artifacts"`
	Environment    Environment `json:"environment"`
}

// TestSummary 测试摘要
type TestSummary struct {
	TotalTests   int     `json:"total_tests"`
	PassedTests  int     `json:"passed_tests"`
	FailedTests  int     `json:"failed_tests"`
	SkippedTests int     `json:"skipped_tests"`
	PassRate     float64 `json:"pass_rate"`
	Duration     string  `json:"duration"`
	Status       string  `json:"status"` // "passed", "failed", "partial"
}

// TestSuite 测试套件
type TestSuite struct {
	Name         string     `json:"name"`
	Package      string     `json:"package"`
	TotalTests   int        `json:"total_tests"`
	PassedTests  int        `json:"passed_tests"`
	FailedTests  int        `json:"failed_tests"`
	SkippedTests int        `json:"skipped_tests"`
	Duration     string     `json:"duration"`
	TestCases    []TestCase `json:"test_cases"`
	Status       string     `json:"status"`
}

// TestCase 测试用例
type TestCase struct {
	Name     string `json:"name"`
	Status   string `json:"status"` // "PASS", "FAIL", "SKIP"
	Duration string `json:"duration"`
	Message  string `json:"message,omitempty"`
}

// BuildInfo 构建信息
type BuildInfo struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
}

// Environment 测试环境
type Environment struct {
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Hostname string `json:"hostname"`
	Socat    string `json:"socat"`
}

// ReportGenerator 报告生成器
type ReportGenerator struct {
	projectRoot string
	reportDir   string
}

// NewReportGenerator 创建报告生成器
func NewReportGenerator(projectRoot string) *ReportGenerator {
	reportDir := filepath.Join(projectRoot, "test-reports")
	os.MkdirAll(reportDir, 0755)
	return &ReportGenerator{
		projectRoot: projectRoot,
		reportDir:   reportDir,
	}
}

// ParseTestLog 解析测试日志
func (rg *ReportGenerator) ParseTestLog(logPath string) (*TestSuite, error) {
	content, err := ioutil.ReadFile(logPath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	suite := &TestSuite{
		Package:      strings.TrimSuffix(filepath.Base(logPath), ".log"),
		TestCases:    make([]TestCase, 0),
		PassedTests:  0,
		FailedTests:  0,
		SkippedTests: 0,
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "=== RUN") {
			// 新测试开始
			name := strings.TrimPrefix(line, "=== RUN   ")
			suite.TestCases = append(suite.TestCases, TestCase{
				Name:   name,
				Status: "RUNNING",
			})
			suite.TotalTests++
		} else if strings.HasPrefix(line, "--- PASS") {
			// 测试通过
			suite.TestCases[len(suite.TestCases)-1].Status = "PASS"
			suite.TestCases[len(suite.TestCases)-1].Duration = extractDuration(line)
			suite.PassedTests++
		} else if strings.HasPrefix(line, "--- FAIL") {
			// 测试失败
			suite.TestCases[len(suite.TestCases)-1].Status = "FAIL"
			suite.TestCases[len(suite.TestCases)-1].Duration = extractDuration(line)
			suite.FailedTests++
		} else if strings.HasPrefix(line, "--- SKIP") {
			// 测试跳过
			suite.TestCases[len(suite.TestCases)-1].Status = "SKIP"
			suite.TestCases[len(suite.TestCases)-1].Duration = extractDuration(line)
			suite.SkippedTests++
		} else if strings.HasPrefix(line, "FAIL") || strings.HasPrefix(line, "PASS") {
			// 套件结果
			if strings.HasPrefix(line, "FAIL") {
				suite.Status = "FAIL"
				suite.Name = strings.TrimSpace(strings.TrimPrefix(line, "FAIL"))
			} else {
				suite.Status = "PASS"
				suite.Name = strings.TrimSpace(strings.TrimPrefix(line, "PASS"))
			}
		}
	}

	return suite, nil
}

// extractDuration 从行中提取时间
func extractDuration(line string) string {
	// 示例: "--- PASS: TestBasicFlow (0.01s)"
	start := strings.Index(line, "(")
	end := strings.Index(line, ")")
	if start != -1 && end != -1 && end > start {
		return line[start+1 : end]
	}
	return "0s"
}

// GenerateReport 生成测试报告
func (rg *ReportGenerator) GenerateReport() (*TestReport, error) {
	report := &TestReport{
		Timestamp:      time.Now().Format("2006-01-02 15:04:05"),
		TestSuites:     make([]TestSuite, 0),
		BuildArtifacts: make([]BuildInfo, 0),
	}

	// Git 信息
	report.GitCommit = rg.getGitCommit()
	report.GitBranch = rg.getGitBranch()

	// Go 版本
	report.GoVersion = rg.getGoVersion()

	// 环境
	report.Environment = rg.getEnvironment()

	// 解析测试日志
	logFiles := []string{
		"unit_test_output.log",
		"e2e_test_output.log",
		"simulation_test_output.log",
	}

	for _, logFile := range logFiles {
		logPath := filepath.Join(rg.reportDir, logFile)
		if _, err := os.Stat(logPath); err == nil {
			suite, err := rg.ParseTestLog(logPath)
			if err != nil {
				fmt.Printf("Warning: failed to parse %s: %v\n", logFile, err)
				continue
			}
			suite.Name = strings.Replace(logFile, "_output.log", "", 1)
			suite.Name = strings.Replace(suite.Name, "_test", "", 1)
			suite.Name = strings.Title(suite.Name)
			report.TestSuites = append(report.TestSuites, *suite)

			// 累加统计
			report.Summary.TotalTests += suite.TotalTests
			report.Summary.PassedTests += suite.PassedTests
			report.Summary.FailedTests += suite.FailedTests
			report.Summary.SkippedTests += suite.SkippedTests
		}
	}

	// 计算通过率
	if report.Summary.TotalTests > 0 {
		report.Summary.PassRate = float64(report.Summary.PassedTests) / float64(report.Summary.TotalTests) * 100
	}

	// 确定状态
	if report.Summary.FailedTests == 0 {
		if report.Summary.PassedTests > 0 {
			report.Summary.Status = "passed"
		} else {
			report.Summary.Status = "no_tests"
		}
	} else if report.Summary.PassedTests > report.Summary.FailedTests {
		report.Summary.Status = "partial"
	} else {
		report.Summary.Status = "failed"
	}

	// 构建产物
	binDir := filepath.Join(rg.projectRoot, "bin")
	if files, err := ioutil.ReadDir(binDir); err == nil {
		for _, file := range files {
			if !file.IsDir() {
				report.BuildArtifacts = append(report.BuildArtifacts, BuildInfo{
					Name:     file.Name(),
					Path:     filepath.Join("bin", file.Name()),
					Size:     file.Size(),
					Modified: file.ModTime().Format("2006-01-02 15:04:05"),
				})
			}
		}
	}

	return report, nil
}

// 辅助方法
func (rg *ReportGenerator) getGitCommit() string {
	// 简化实现
	return "N/A"
}

func (rg *ReportGenerator) getGitBranch() string {
	return "main"
}

func (rg *ReportGenerator) getGoVersion() string {
	return "go1.21"
}

func (rg *ReportGenerator) getEnvironment() Environment {
	return Environment{
		OS:       "linux",
		Arch:     "amd64",
		Hostname: "localhost",
		Socat:    "installed",
	}
}

// SaveMarkdownReport 保存 Markdown 报告
func (rg *ReportGenerator) SaveMarkdownReport(report *TestReport, outputPath string) error {
	tmpl := `# ShareSerial 测试报告

**生成时间:** {{.Timestamp}}

**Git 版本:** {{.GitCommit}} (分支: {{.GitBranch}})

**Go 版本:** {{.GoVersion}}

---

## 测试摘要

| 指标 | 数值 |
|------|------|
| 状态 | {{.Summary.Status}} |
| 总测试数 | {{.Summary.TotalTests}} |
| 通过数 | {{.Summary.PassedTests}} |
| 失败数 | {{.Summary.FailedTests}} |
| 跳过数 | {{.Summary.SkippedTests}} |
| 通过率 | {{printf "%.2f" .Summary.PassRate}}% |

---

## 测试套件详情

{{range .TestSuites}}
### {{.Name}}

- **状态:** {{.Status}}
- **测试数:** {{.TotalTests}} (通过: {{.PassedTests}}, 失败: {{.FailedTests}}, 跳过: {{.SkippedTests}})
- **耗时:** {{.Duration}}

| 测试名称 | 状态 | 耗时 |
|---------|------|------|
{{range .TestCases}}| {{.Name}} | {{.Status}} | {{.Duration}} |
{{end}}

{{end}}

---

## 构建产物

| 文件名 | 路径 | 大小 | 修改时间 |
|--------|------|------|----------|
{{range .BuildArtifacts}}| {{.Name}} | {{.Path}} | {{.Size}} bytes | {{.Modified}} |
{{end}}

---

## 测试环境

- **操作系统:** {{.Environment.OS}}
- **架构:** {{.Environment.Arch}}
- **主机名:** {{.Environment.Hostname}}
- **socat:** {{.Environment.Socat}}

---

*报告生成时间: {{.Timestamp}}*
`

	t, err := template.New("report").Parse(tmpl)
	if err != nil {
		return err
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return t.Execute(file, report)
}

// SaveHTMLReport 保存 HTML 报告
func (rg *ReportGenerator) SaveHTMLReport(report *TestReport, outputPath string) error {
	htmlTmpl := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>ShareSerial 测试报告</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 2px solid #4CAF50; padding-bottom: 10px; }
        h2 { color: #666; margin-top: 30px; }
        h3 { color: #444; margin-top: 20px; }
        table { width: 100%; border-collapse: collapse; margin-top: 15px; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background: #f8f9fa; font-weight: 600; }
        .status-passed { color: #4CAF50; font-weight: bold; }
        .status-failed { color: #f44336; font-weight: bold; }
        .status-partial { color: #FF9800; font-weight: bold; }
        .status-no_tests { color: #9E9E9E; }
        .badge { padding: 4px 8px; border-radius: 4px; font-size: 12px; }
        .badge-pass { background: #E8F5E9; color: #2E7D32; }
        .badge-fail { background: #FFEBEE; color: #C62828; }
        .badge-skip { background: #FFF3E0; color: #EF6C00; }
        .summary-box { background: #f8f9fa; padding: 20px; border-radius: 8px; margin-bottom: 20px; }
        .summary-box h2 { margin-top: 0; }
        .metric { display: inline-block; margin-right: 30px; }
        .metric-label { color: #666; font-size: 14px; }
        .metric-value { color: #333; font-size: 24px; font-weight: bold; }
        .timestamp { color: #666; font-size: 14px; }
        footer { margin-top: 30px; padding-top: 20px; border-top: 1px solid #ddd; color: #666; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>ShareSerial 测试报告</h1>
        <p class="timestamp">生成时间: {{.Timestamp}} | Git: {{.GitCommit}} ({{.GitBranch}}) | Go: {{.GoVersion}}</p>

        <div class="summary-box">
            <h2>测试摘要</h2>
            <div class="metric">
                <div class="metric-label">状态</div>
                <div class="metric-value status-{{.Summary.Status}}">{{.Summary.Status}}</div>
            </div>
            <div class="metric">
                <div class="metric-label">总测试数</div>
                <div class="metric-value">{{.Summary.TotalTests}}</div>
            </div>
            <div class="metric">
                <div class="metric-label">通过数</div>
                <div class="metric-value">{{.Summary.PassedTests}}</div>
            </div>
            <div class="metric">
                <div class="metric-label">失败数</div>
                <div class="metric-value">{{.Summary.FailedTests}}</div>
            </div>
            <div class="metric">
                <div class="metric-label">通过率</div>
                <div class="metric-value">{{printf "%.2f" .Summary.PassRate}}%</div>
            </div>
        </div>

        <h2>测试套件详情</h2>
        {{range .TestSuites}}
        <h3>{{.Name}} <span class="status-{{.Status}}">({{.Status}})</span></h3>
        <p>测试数: {{.TotalTests}} | 通过: {{.PassedTests}} | 失败: {{.FailedTests}} | 跳过: {{.SkippedTests}}</p>
        <table>
            <thead>
                <tr>
                    <th>测试名称</th>
                    <th>状态</th>
                    <th>耗时</th>
                </tr>
            </thead>
            <tbody>
                {{range .TestCases}}
                <tr>
                    <td>{{.Name}}</td>
                    <td><span class="badge badge-{{.Status | lower}}">{{.Status}}</span></td>
                    <td>{{.Duration}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>
        {{end}}

        <h2>构建产物</h2>
        <table>
            <thead>
                <tr>
                    <th>文件名</th>
                    <th>路径</th>
                    <th>大小</th>
                    <th>修改时间</th>
                </tr>
            </thead>
            <tbody>
                {{range .BuildArtifacts}}
                <tr>
                    <td>{{.Name}}</td>
                    <td>{{.Path}}</td>
                    <td>{{.Size}} bytes</td>
                    <td>{{.Modified}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>

        <h2>测试环境</h2>
        <table>
            <tr><th>操作系统</th><td>{{.Environment.OS}}</td></tr>
            <tr><th>架构</th><td>{{.Environment.Arch}}</td></tr>
            <tr><th>主机名</th><td>{{.Environment.Hostname}}</td></tr>
            <tr><th>socat</th><td>{{.Environment.Socat}}</td></tr>
        </table>

        <footer>
            <p>报告生成时间: {{.Timestamp}}</p>
        </footer>
    </div>
</body>
</html>
`

	// 添加 lower 函数
	t, err := template.New("html").Funcs(template.FuncMap{
		"lower": strings.ToLower,
	}).Parse(htmlTmpl)
	if err != nil {
		return err
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return t.Execute(file, report)
}

// SaveJSONReport 保存 JSON 报告
func (rg *ReportGenerator) SaveJSONReport(report *TestReport, outputPath string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(outputPath, data, 0644)
}

func main() {
	// 从命令行参数获取项目根目录
	projectRoot := "."
	if len(os.Args) > 1 {
		projectRoot = os.Args[1]
	}

	rg := NewReportGenerator(projectRoot)

	report, err := rg.GenerateReport()
	if err != nil {
		fmt.Printf("Error generating report: %v\n", err)
		os.Exit(1)
	}

	// 生成时间戳
	timestamp := time.Now().Format("20060102_150405")

	// 保存多种格式的报告
	rg.SaveMarkdownReport(report, filepath.Join(rg.reportDir, fmt.Sprintf("report_%s.md", timestamp)))
	rg.SaveHTMLReport(report, filepath.Join(rg.reportDir, fmt.Sprintf("report_%s.html", timestamp)))
	rg.SaveJSONReport(report, filepath.Join(rg.reportDir, fmt.Sprintf("report_%s.json", timestamp)))

	fmt.Println("Reports generated:")
	fmt.Printf("  Markdown: test-reports/report_%s.md\n", timestamp)
	fmt.Printf("  HTML:     test-reports/report_%s.html\n", timestamp)
	fmt.Printf("  JSON:     test-reports/report_%s.json\n", timestamp)
	fmt.Printf("\nSummary: %d tests, %d passed, %d failed, %.2f%% pass rate\n",
		report.Summary.TotalTests,
		report.Summary.PassedTests,
		report.Summary.FailedTests,
		report.Summary.PassRate)
}
