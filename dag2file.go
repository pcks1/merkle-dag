package merkledag

import (
	"encoding/json"
	"errors"
)

// Hash2File 根据 hash 和 path 从 KVStore 中读取并返回对应的文件内容。
func Hash2File(store KVStore, hash []byte, path string) ([]byte, error) {
	// 从 KVStore 中获取 Node 对象
	nodeData, err := store.Get(hash)
	if err != nil {
		return nil, err
	}

	// 解析 Node 对象
	var node Node
	if err := json.Unmarshal(nodeData, &node); err != nil {
		return nil, err
	}

	// 根据 Node 类型处理
	switch node.(type) {
	case File:
		// 如果是文件类型，直接返回文件内容
		file := node.(File)
		if file.Name() == path {
			return file.Bytes(), nil
		}
	case Dir:
		// 如果是目录类型，遍历目录
		dir := node.(Dir)
		iterator := dir.It()
		for iterator.Next() {
			currentNode := iterator.Node()
			// 如果路径匹配，且节点为文件类型，返回文件内容
			if currentNode.Name() == path && currentNode.Type() == FILE {
				file, ok := currentNode.(File)
				if !ok {
					return nil, errors.New("iterator node type assertion to File failed")
				}
				return file.Bytes(), nil
			}
		}
	}

	return nil, errors.New("file not found")
}
