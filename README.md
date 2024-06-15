# kv储存引擎兼容redis
直接采用现有的开源 KV 存储引擎（badger）
代码位于badger的main中

# 代码的总结：
1.  **定义BadgerStore结构体**：
    *BadgerStore 结构体包含一个Badger数据库实例和一个用户对象。
    *实现Set、Get、Delete方法
2.   **简单的用户权限**:定义了简单的用户权限一个简单的用户权限系统，它使用枚举类型来表示不同的权限，并通过 User 结构体和 BadgerStore 结构体中的方法来检查和验证权限。
3.  **主函数**：
    *   使用Badger数据库进行一系列操作，包括设置、获取和删除键值对。
    *   使用 BadgerDB（一个键值存储数据库）来模拟 Redis 的常见数据结构，如字符串、列表、集合、哈希表和有序映射。代码中包含了创建数据库、设置值、获取值、删除键、存储和读取不同数据结构的方法。
    *   将Badger数据库封装成一个兼容Redis的存储系统，实现了使用Redis客户端操作Badger数据库的功能
    *   TCP服务器，它监听在端口6379上，并使用BadgerStore来处理Redis命令。同时，尝试建立一个到Redis服务器的TCP连接。
4. **处理Redis命令**：
    *   handleConnection 和 BadgerStore.processCommands。这两个函数共同处理来自客户端的Redis命令。
# 具体功能
1.  **BadgerStore 结构体和方法**：
    *   `BadgerStore` 结构体包含一个Badger数据库实例和一个用户对象。
    *   实现了`Set`、`Get`、`Delete`方法，用于处理键值对的存储、读取和删除操作。这些方法还包括权限检查，确保用户有权执行相应操作。
2.  **用户和权限管理**：
    *   `User` 结构体表示用户，包含用户名和权限映射。
    *   `Permission` 是一个枚举类型，定义了读、写、删除等权限。
    *   `HasPermission` 方法用于检查用户是否具有特定权限。
3.  **Redis命令处理**：
    *   `handleConnection` 函数处理TCP连接，读取客户端发送的命令，并根据命令类型（如SET、GET）执行相应的操作。
    *   `processCommands` 方法处理Redis命令，检查权限并执行数据库操作。
4.  **数据结构兼容性**：
    *   代码展示了如何使用Badger数据库模拟Redis的数据结构，如字符串、列表、哈希表、集合和有序映射。
    *   通过序列化和反序列化JSON，实现了复杂数据结构的存储和读取。
5.  **Redis客户端兼容性**：
    *   创建了一个Redis客户端，并将其配置为使用BadgerStore作为后端存储。
    *   通过`WrapProcess`方法，将BadgerStore的命令处理逻辑集成到Redis客户端中。
6.  **网络和命令行交互**：
    *   程序监听TCP端口6379，处理来自客户端的连接和命令。
    *   提供了命令行交互示例，展示了如何使用Redis客户端与BadgerStore进行交互。


