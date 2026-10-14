package probe

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/pkg/errors"
)

func marshalResult(r Result) ([]byte, error) {
	line, err := json.Marshal(r)
	if err != nil {
		return nil, errors.Wrap(err, "序列化结果失败")
	}
	return line, nil
}

// Store 以 JSON Lines 格式（每行一条结果）追加存储拨测结果，便于后续分析。
type Store struct {
	mu   sync.Mutex
	file *os.File
}

// NewStore 打开（不存在则创建）结果文件，采用追加写入。
func NewStore(path string) (*Store, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, errors.Wrapf(err, "打开结果文件失败: %s", path)
	}
	return &Store{file: f}, nil
}

// Append 将一批结果逐行写入文件并落盘。
func (s *Store) Append(results []Result) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, r := range results {
		line, err := marshalResult(r)
		if err != nil {
			return err
		}
		if _, err := s.file.Write(append(line, '\n')); err != nil {
			return errors.Wrap(err, "写入结果失败")
		}
	}
	return s.file.Sync()
}

// Close 关闭结果文件。
func (s *Store) Close() error {
	if s.file == nil {
		return nil
	}
	return s.file.Close()
}
