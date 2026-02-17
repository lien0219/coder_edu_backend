package service

import (
	"bufio"
	"bytes"
	"coder_edu_backend/internal/config"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type AIService struct {
	config config.AIConfig
}

func NewAIService(cfg config.AIConfig) *AIService {
	return &AIService{config: cfg}
}

type AIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model    string          `json:"model"`
	Messages []AIChatMessage `json:"messages"`
}

type ChatCompletionResponse struct {
	Choices []struct {
		Message AIChatMessage `json:"message"`
		Delta   AIChatMessage `json:"delta"` // 流式响应
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (s *AIService) ChatStream(prompt string, context string) (<-chan string, <-chan error) {
	out := make(chan string)
	errChan := make(chan error, 1)

	// 注入Markdown格式规范，确保与前端防御性渲染器完美配合
	markdownGuideline := "\n\n【Markdown 渲染指令 - 必须严格执行，否则前端无法解析】\n" +
		"1. 代码块独立成行（核心）：\n" +
		"   - 在输出 ```c 之前，必须先输出两个换行符（\\n\\n）。\n" +
		"   - 在 ```c 之后，必须立即输出一个换行符（\\n），严禁在三反引号同一行写代码内容。\n" +
		"   - 错误示例：## 基础实现 ```c #include\n" +
		"   - 正确示例：## 基础实现 \\n\\n ```c \\n #include\n" +
		"2. 标题物理分隔：## 标题之后必须紧跟两个换行符（\\n\\n）再开始正文或代码块。\n" +
		"3. 代码注释规范：代码内部的注释（//）与代码行之间必须保持正常的换行，严禁为了节省空间而将注释与代码挤在同一行。\n" +
		"4. 三反引号闭合：代码结束后的 ``` 必须独占一行，且其后必须紧跟两个换行符（\\n\\n）。\n" +
		"5. 严禁粘连：严禁将标题、正文、代码块这三者中的任何两个放在同一行输出。"

	messages := []AIChatMessage{}
	if context != "" {
		messages = append(messages, AIChatMessage{
			Role:    "system",
			Content: fmt.Sprintf("你是一个教育助教。请结合以下背景知识回答问题：\n\n%s%s", context, markdownGuideline),
		})
	} else {
		messages = append(messages, AIChatMessage{
			Role:    "system",
			Content: "你是一个专业的编程教育助教，请尽力回答学生的问题。" + markdownGuideline,
		})
	}

	messages = append(messages, AIChatMessage{
		Role:    "user",
		Content: prompt,
	})

	reqBody := map[string]interface{}{
		"model":    s.config.Model,
		"messages": messages,
		"stream":   true,
	}

	jsonData, _ := json.Marshal(reqBody)

	go func() {
		defer close(out)
		defer close(errChan)

		req, err := http.NewRequest("POST", s.config.BaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
		if err != nil {
			errChan <- err
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+s.config.APIKey)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			errChan <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errChan <- fmt.Errorf("AI API error (status %d): %s", resp.StatusCode, string(body))
			return
		}

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					errChan <- err
				}
				break
			}

			line = strings.TrimSpace(line)
			if line == "" || !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var streamResp ChatCompletionResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				continue
			}

			if len(streamResp.Choices) > 0 {
				content := streamResp.Choices[0].Delta.Content
				if content != "" {
					out <- content
				}
			}
		}
	}()

	return out, errChan
}

func (s *AIService) Chat(prompt string, context string) (string, error) {
	messages := []AIChatMessage{}

	if context != "" {
		messages = append(messages, AIChatMessage{
			Role:    "system",
			Content: fmt.Sprintf("你是一个教育助教。请结合以下背景知识回答问题：\n\n%s", context),
		})
	} else {
		messages = append(messages, AIChatMessage{
			Role:    "system",
			Content: "你是一个专业的编程教育助教，请尽力回答学生的问题。",
		})
	}

	messages = append(messages, AIChatMessage{
		Role:    "user",
		Content: prompt,
	})

	reqBody := ChatCompletionRequest{
		Model:    s.config.Model,
		Messages: messages,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", s.config.BaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.config.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AI API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result ChatCompletionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("AI returned no choices")
}
