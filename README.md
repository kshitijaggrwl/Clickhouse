# High-Performance Data Serialization in Golang

##  Overview
This project provides a **highly optimized** data serialization format for efficient communication between a client and a database server (e.g., ClickHouse).

##  Supported Data Types
- **String (`string`)** – Supports UTF-8 characters (max length: `1,000,000`).
- **Integer (`int32`)** – 32-bit signed integers.
- **Floating Point (`float64`)** – IEEE 754 double-precision floating point numbers.
- **Nested Arrays (`DataInput`)** – Supports recursive data structures (max depth: **1000**).


##  Time & Space Complexity Analysis
### **Encoding (`encode`)**
| Operation | Time Complexity | Space Complexity |
|-----------|----------------|------------------|
| Encoding `string` (size `n`) | `O(n)` | `O(n)` |
| Encoding `int32` | `O(1)` | `O(4 bytes)` |
| Encoding `float64` | `O(1)` | `O(8 bytes)` |
| Encoding `DataInput` (size `m`) | `O(m)` | `O(m)` |
| Total Complexity | `O(N)` | `O(N)` |

### **Decoding (`decode`)**
| Operation | Time Complexity | Space Complexity |
|-----------|----------------|------------------|
| Decoding `string` (size `n`) | `O(n)` | `O(n)` |
| Decoding `int32` | `O(1)` | `O(4 bytes)` |
| Decoding `float64` | `O(1)` | `O(8 bytes)` |
| Decoding `DataInput` (size `m`) | `O(m)` | `O(m)` |
| Total Complexity | `O(N)` | `O(N)` |


##  Optimizations for Speed & Memory
###  Memory Pooling (`sync.Pool`)
- **Why?** Avoid unnecessary memory allocations.
- **How?** Buffers are **reused** instead of allocating new ones each time.

###  Compact Binary Format
- **Why?** Reduces transmission time & storage footprint.
- **How?** Uses **Varint Encoding** for efficient integer representation.

###  Zero-Copy String Conversion (`unsafe.Pointer`)
- **Why?** Avoids extra memory copying.
- **How?** Converts `[]byte` to `string` **without additional allocations**.


##  How to Add Support for More Data Types
###  **1️⃣ Modify the `encode` Function**
Add a new case in the `switch` statement to handle the new type. Example:
```go
case bool:
    buf = append(buf, 'B') // 'B' for Boolean
    if v {
        buf = append(buf, 1)
    } else {
        buf = append(buf, 0)
    }
```

###  **2️⃣ Modify the `decode` Function**
Add logic to recognize and decode the new type:
```go
case 'B': // Boolean
    *pos++
    if data[*pos] == 1 {
        result = append(result, true)
    } else {
        result = append(result, false)
    }
    *pos++
```


