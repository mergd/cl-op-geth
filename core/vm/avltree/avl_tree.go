package vm

import (
	"database/sql"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	_ "github.com/mattn/go-sqlite3"
)

type Pair struct {
	pairId         int32 // pair ID
	asset0         uint32
	asset1         uint32
	tickSpacing    uint16 // tick spacing for the market
	tickLowerBound uint32 // lower bound of the tick range
	tickUpperBound uint32 // upper bound of the tick range
}

type Node struct {
	key    uint32
	height uint32
	left   *Node
	right  *Node
	weight uint256.Int // Sum of the orders in the subtree
}

type Order struct {
	id     uint64
	pairId int32  // positive for tkn0 -> tkn1, negative  for tkn1 -> tkn0
	price  uint32 // price of the order (token0/token1) with tickSpacing
	amount uint256.Int
	from   common.Address
	// deadline uint64 // Deadline for order execution
}

var userOrders map[common.Address]map[int32][]Order
var orderBook map[int32]*Node
var paircount uint32

func initialize() {
	// userOrders = make(map[common.Address][0][]Order)
	initializeDatabase()
}

/* -------------------------------------------------------------------------- */
/*                               Initialization                               */
/* -------------------------------------------------------------------------- */

func initializeDatabase() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./nodes.db")
	if err != nil {
		return nil, err
	}

	statement, err := db.Prepare("CREATE TABLE IF NOT EXISTS nodes (key INTEGER, height INTEGER, weight TEXT)")
	if err != nil {
		return nil, err
	}

	_, err = statement.Exec()
	if err != nil {
		return nil, err
	}

	return db, nil
}

/* -------------------------------------------------------------------------- */
/*                                    Utils                                   */
/* -------------------------------------------------------------------------- */

func height(node *Node) uint32 {
	if node == nil {
		return 0
	}
	return node.height
}

func max(a, b uint32) uint32 {
	if a > b {
		return a
	}
	return b
}

/* -------------------------------------------------------------------------- */
/*                                  Tree Ops                                  */
/* -------------------------------------------------------------------------- */

func createPair(pair Pair) uint32 {
	// Create a new pair
	paircount++
	orderBook[int32(paircount)] = nil
	orderBook[int32(-paircount)] = nil
	return paircount
}

func addOrder(order Order) *Node {
	node := orderBook[order.pairId]

	if node == nil {
		// Initialize new node
		node = &Node{key: order.price, height: 1, left: nil, right: nil, weight: order.amount}
	} else {
		// add the node to the tree
		node = node._insertNode(order)
	}

	// Add order to userOrders for this pair
	// Todo add a max size to prevent ddos
	userOrders[order.from][order.pairId] = append(userOrders[order.from][order.pairId], order)
	return node
}

func (node *Node) _insertNode(order Order) *Node {
	// add weight to the node
	node.weight.Add(&node.weight, &order.amount)
	if order.price < node.key {
		node.left = node._insertNode(order)
	} else if order.price > node.key {
		node.right = node._insertNode(order)
	} else {
		return node
	}
	node.height = max(height(node.left), height(node.right)) + 1
	return node
}

// func deleteNode(key uint32, amount uint256.Int) {
// 	// Delete the order from the userOrders
// 	orders := userOrders[msg.sender]
// }

func deleteOrder(pairId int32, order Order) {
	// Delete the order from the userOrders
	orders := userOrders[order.from][pairId]
	for i, order := range orders {
		if order.id == order.id {
			orders = append(orders[:i], orders[i+1:]...)
			break
		}
	}
	// Delete the order from the orderBook
	_deleteNode(orderBook[order.pairId], order.price, order.amount)
}

func deleteUserOrder(pairId int32, user common.Address, orderId uint64) {
	// Delete the order from the userOrders
	orders := userOrders[user][pairId]
	for i, order := range orders {
		if order.id == orderId {
			orders = append(orders[:i], orders[i+1:]...)
			break
		}
	}
}

// func  viewNode(pairId int32, key uint32) *Node {
// 	// View the depth for the specified node
// }

func viewOrders(pairIds int32, user common.Address) []Order {
	return userOrders[user][pairIds]
}

func balanceTree(root Node) Node {
	// todo balance tree after it's disrupted
	return root
}

// todo
func _deleteNode(root *Node, key uint32, amount uint256.Int) (*Node, uint256.Int) {
	// Market not intialized
	if root == nil {
		return root, *uint256.NewInt(0)
	}

	// Order limit price surpassed
	if key > root.key {
		return root, uint256.Int{0}
	}

	// Base Case pt2
	if root.left == nil && root.right == nil && !amount.IsZero() {

		//  if zero, delete the node as parent node is cleared
		// If the node is a leaf node, delete it and return the weight
		// return nil, root.weight
		if root.weight.Lt(&amount) {

		}
	}
	// If amount is gt than the weight, delete the node – this is the base case pt 1
	if root.weight.Lt(&amount) {
		root.weight = uint256.Int{0}
		// Clear children root nodes
		if root.right != nil {
			_deleteNode(root.right, key, amount)
		}
		if root.left != nil {
			_deleteNode(root.left, key, amount)
		}
		return root, root.weight
	}

	// If left is gt than amt – recurse
	if root.left.weight.Lt(&amount) {
		_, value := _deleteNode(root.left, key, amount)
		root.weight.Sub(&root.weight, &value)

	} else if root.right.weight.Gt(&amount) {
		newAmt := *amount.Sub(&amount, &root.left.weight)
		_deleteNode(root.left, key, amount)
		_, value := _deleteNode(root.right, key, newAmt)

		// Update the weight of the root
		root.weight.Sub(&root.weight, newAmt.Add(&newAmt, &value))
	}

	// Move down if the amount is less than the weight
	if amount.Lt(&root.weight) {
		left, weightChg := _deleteNode(root.left, key, amount)

		root.left = left
		root.weight.Sub(&root.weight, &weightChg)
	} else {
		// Delete the next node
		return root, uint256.Int{0}
	}
	return root, uint256.Int{0}
}
