package merkledag

import (
	"encoding/binary"
	"encoding/json"
	"hash"
)

const (
	K          = 1 << 10
	BLOCK_SIZE = 256 * K
)

type Link struct {
	Name string
	Hash []byte
	Size int
}

type Object struct {
	Links []Link
	Data  [][]byte
}

var Stack = make([]Object, 0)

type ListNode struct {
	Hash []byte
	Next []byte
}

func (n *ListNode) Bytes() []byte {
	// 创建一个足够大的缓冲区来存储Hash和Next
	buf := make([]byte, len(n.Hash)+len(n.Next)+16)
	// 将Hash的长度和内容写入到缓冲区
	binary.BigEndian.PutUint64(buf, uint64(len(n.Hash)))
	copy(buf[8:], n.Hash)
	// 将Next的长度和内容写入到缓冲区
	binary.BigEndian.PutUint64(buf[8+len(n.Hash):], uint64(len(n.Next)))
	copy(buf[16+len(n.Hash):], n.Next)
	return buf
}
func Add(store KVStore, node Node, h hash.Hash) []byte {
	switch node.Type() {
	case FILE:
		file := node.(File)
		fileHash, data := StoreFile(store, file, h)
		// 从栈顶取出Object对象
		obj := Stack[len(Stack)-1]
		obj.Links = append(obj.Links, Link{
			Name: file.Name(),
			Hash: fileHash,
			Size: int(file.Size()),
		})
		obj.Data = append(obj.Data, data)
	case DIR:
		dir := node.(Dir)
		// 创建一个新的Object对象并添加到栈顶
		obj := Object{
			Links: make([]Link, 0),
			Data:  make([][]byte, 0),
		}
		Stack = append(Stack, obj)
		// 递归调用Add方法
		dirHash, data := StoreDir(store, dir, h)
		// 从栈顶取出Object对象
		obj = Stack[len(Stack)-1]
		Stack = Stack[:len(Stack)-1]
		obj.Links = append(obj.Links, Link{
			Name: dir.Name(),
			Hash: dirHash,
			Size: int(dir.Size()),
		})
		obj.Data = append(obj.Data, data)
		// 计算Object对象的hash
		h.Reset()
		objBytes, _ := json.Marshal(obj)
		h.Write(objBytes)
		objHash := h.Sum(nil)
		// 将Object对象写入到KVStore中
		store.Put(objHash, objBytes)
		return objHash
	}
	return nil
}

func StoreFile(store KVStore, node File, h hash.Hash) ([]byte, []byte) {
	t := []byte("blob")
	if node.Size() > BLOCK_SIZE {
		t = []byte("list")
	}
	// 计算分片数量
	n := (node.Size() + BLOCK_SIZE - 1) / BLOCK_SIZE
	// 初始化链表头节点的hash为nil
	var headHash []byte = nil
	for i := 0; i < int(n); i++ {
		// 读取分片数据
		data := node.Bytes()[uint64(i*BLOCK_SIZE):uint64((i+1)*BLOCK_SIZE)]
		// 计算分片的hash
		h.Reset()
		h.Write(data)
		fileHash := h.Sum(nil)
		// 将分片写入到KVStore中
		store.Put(fileHash, data)
		// 创建链表节点，包含当前分片的hash和头节点的hash
		listNode := ListNode{Hash: fileHash, Next: headHash}
		// 计算链表节点的hash
		h.Reset()
		h.Write(listNode.Bytes())
		listNodeHash := h.Sum(nil)
		// 将链表节点写入到KVStore中
		store.Put(listNodeHash, listNode.Bytes())
		// 更新链表头节点的hash
		headHash = listNodeHash
	}
	return headHash, t
}

func StoreDir(store KVStore, dir Dir, h hash.Hash) ([]byte, []byte) {
	t := []byte("tree")
	tree := Object{
		Links: make([]Link, 0),
		Data:  make([][]byte, 0),
	}
	it := dir.It()
	for it.Next() {
		node := it.Node()
		switch node.Type() {
		case FILE:
			file := node.(File)
			fileHash, _ := StoreFile(store, file, h)
			tree.Links = append(tree.Links, Link{
				Name: file.Name(),
				Hash: fileHash,
				Size: int(file.Size()),
			})
		case DIR:
			subDir := node.(Dir)
			subDirHash, _ := StoreDir(store, subDir, h)
			tree.Links = append(tree.Links, Link{
				Name: subDir.Name(),
				Hash: subDirHash,
				Size: int(subDir.Size()),
			})
		}
	}
	// 计算tree对象的hash
	h.Reset()
	treeBytes, _ := json.Marshal(tree)
	h.Write(treeBytes)
	treeHash := h.Sum(nil)
	// 将tree对象写入到KVStore中
	err := store.Put(treeHash, treeBytes)
	if err != nil {
		return nil, nil
	}
	return treeHash, t
}
