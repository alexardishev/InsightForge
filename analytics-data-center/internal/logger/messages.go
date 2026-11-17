package logger

// Predefined multilingual messages
var (
	MsgAnalyticsServerStart = Message{
		RU: "Запуск сервера аналитики",
		EN: "Starting Analytics server",
		CN: "启动分析服务器",
	}
	MsgHTTPServerStarted = Message{
		RU: "HTTP сервер запущен на порту 8888",
		EN: "HTTP server started on port 8888",
		CN: "HTTP服务器在8888端口启动",
	}
	MsgStoppingApplication = Message{
		RU: "Остановка приложения",
		EN: "stopping application",
		CN: "停止应用",
	}
	MsgApplicationStopped = Message{
		RU: "Приложение остановлено корректно",
		EN: "Application stopped gracefully",
		CN: "应用已正常停止",
	}

	MsgNoChanges = Message{
		RU: "миграции не требуются",
		EN: "no changes migrates",
		CN: "无需迁移",
	}

	MsgMigrationsCompleted = Message{
		RU: "миграции завершены",
		EN: "Migrations is completed",
		CN: "迁移已完成",
	}

	MsgCDCMessageReceived = Message{
		RU: "CDC сообщение получено",
		EN: "CDC message received",
		CN: "收到CDC消息",
	}
	MsgKafkaCommitError = Message{
		RU: "Ошибка при коммите Kafka offset",
		EN: "Kafka commit offset error",
		CN: "提交Kafka偏移量错误",
	}
	MsgKafkaError = Message{
		RU: "Kafka ошибка",
		EN: "Kafka error",
		CN: "Kafka错误",
	}

	// CDC listener messages
	MsgCDCHandlerFailed        = Message{RU: "Не удалось обработать CDC-событие, будет повтор", EN: "CDC event processing failed, will retry", CN: "CDC事件处理失败，将重试"}
	MsgCDCRetryScheduled       = Message{RU: "Партиция поставлена на паузу до повтора", EN: "partition paused until retry", CN: "分区已暂停，等待重试"}
	MsgCDCResumeFailed         = Message{RU: "Не удалось снять паузу с партиции", EN: "failed to resume partition", CN: "恢复分区失败"}
	MsgCDCPermanentError       = Message{RU: "Постоянная ошибка CDC-события, отправляю в DLQ", EN: "permanent CDC event error, sending to DLQ", CN: "CDC事件永久错误，发送到DLQ"}
	MsgCDCSentToDLQ            = Message{RU: "CDC-событие отправлено в DLQ, offset закоммичен", EN: "CDC event sent to DLQ, offset committed", CN: "CDC事件已发送到DLQ"}
	MsgCDCDLQSendFailed        = Message{RU: "Не удалось отправить событие в DLQ, будет повтор", EN: "failed to send event to DLQ, will retry", CN: "发送到DLQ失败，将重试"}
	MsgCDCTombstoneSkipped     = Message{RU: "Пустое сообщение (tombstone) пропущено", EN: "tombstone message skipped", CN: "已跳过空消息(tombstone)"}
	MsgCDCStopped              = Message{RU: "CDC остановлен с ошибкой, завершаю приложение", EN: "CDC stopped with error, shutting down", CN: "CDC出错停止，正在关闭应用"}
	MsgCDCShutdownTimeout      = Message{RU: "CDC не остановился за отведённое время", EN: "CDC did not stop in time", CN: "CDC未能按时停止"}
	MsgKafkaInitFailed         = Message{RU: "Не удалось инициализировать Kafka", EN: "Kafka initialization failed", CN: "Kafka初始化失败"}
	MsgKafkaCloseFailed        = Message{RU: "Ошибка при закрытии Kafka", EN: "Kafka close failed", CN: "关闭Kafka失败"}
	MsgKafkaDLQEventError      = Message{RU: "Ошибка продюсера DLQ", EN: "DLQ producer error", CN: "DLQ生产者错误"}
	MsgKafkaDLQTopicFailed     = Message{RU: "Не удалось создать DLQ-топик", EN: "failed to create DLQ topic", CN: "创建DLQ主题失败"}
	MsgKafkaDLQRetentionSet    = Message{RU: "Срок хранения DLQ-топика приведён к конфигу", EN: "DLQ topic retention set from config", CN: "已按配置设置DLQ主题保留时间"}
	MsgKafkaDLQRetentionFailed = Message{RU: "Не удалось изменить срок хранения DLQ-топика", EN: "failed to set DLQ topic retention", CN: "设置DLQ主题保留时间失败"}

	// Tasks service messages
	MsgCreateTaskStart    = Message{RU: "Создание задачи", EN: "Create task start", CN: "开始创建任务"}
	MsgCreateTaskFailed   = Message{RU: "не удалось создать задачу", EN: "failed to create task", CN: "创建任务失败"}
	MsgChangeStatusStart  = Message{RU: "Изменение статуса задачи", EN: "Change task status", CN: "更改任务状态"}
	MsgChangeStatusFailed = Message{RU: "не изменить статус у задачи", EN: "failed to change task status", CN: "无法更改任务状态"}
	MsgGetTaskStart       = Message{RU: "Получение задачи", EN: "Get task start", CN: "开始获取任务"}
	MsgGetTaskFailed      = Message{RU: "не удалось получить задачу", EN: "failed to get task", CN: "获取任务失败"}

	// Analytics service messages
	MsgETLWorkerStart          = Message{RU: "Начало обработки задачи", EN: "task processing start", CN: "开始处理任务"}
	MsgETLStart                = Message{RU: "ETL запущен", EN: "ETL start", CN: "ETL开始"}
	MsgGenerateQueriesFailed   = Message{RU: "не удалось сгенерировать запросы", EN: "failed to generate queries", CN: "生成查询失败"}
	MsgTempTable               = Message{RU: "Временная таблица", EN: "Temporary table", CN: "临时表"}
	MsgCreateTempTablesFailed  = Message{RU: "не удалось создать временные таблицы", EN: "failed to create temp tables", CN: "创建临时表失败"}
	MsgCountRowsFailed         = Message{RU: "не удалось получить количество", EN: "failed to get count", CN: "获取数量失败"}
	MsgInsertDataFailed        = Message{RU: "не удалось получить данные для вставки", EN: "failed to get insert data", CN: "获取插入数据失败"}
	MsgTransferIndexesFailed   = Message{RU: "не удалось перенести индексы", EN: "failed to transfer indexes", CN: "转移索引失败"}
	MsgEnableReplicationFailed = Message{RU: "не удалось включить полную репликацию", EN: "failed to enable replication", CN: "启用复制失败"}
	MsgReplicationEnabled      = Message{RU: "Репликация для вью включена", EN: "replication enabled", CN: "视图复制已启用"}
	MsgTableRecordCount        = Message{RU: "количество записей в таблице", EN: "table record count", CN: "表记录数"}

	// Analytics event worker messages
	MsgEventWorkerReceived  = Message{RU: "Событие пришло в eventWorker", EN: "event received in worker", CN: "事件已到达worker"}
	MsgEventWorkerError     = Message{RU: "Ошибка при выполнени eventWorker", EN: "event worker error", CN: "worker执行错误"}
	MsgForwardCDCEvent      = Message{RU: "Пришло событие из Kafka, пересылаю в канал", EN: "CDC event received, forwarding", CN: "收到Kafka事件，转发"}
	MsgEventHandlerError    = Message{RU: "Ошибка при определении функции", EN: "handler resolve error", CN: "处理函数解析错误"}
	MsgUpdatesDone          = Message{RU: "Обновления выполнены", EN: "updates completed", CN: "更新完成"}
	MsgDeterminingEventType = Message{RU: "Определяю тип события", EN: "determine event type", CN: "确定事件类型"}
	MsgDeterminingFuncType  = Message{RU: "Определяю тип вызываемой функции", EN: "determine function type", CN: "确定调用函数类型"}

	MsgKafkaAdminClientCreateError = Message{
		RU: "Ошибка при создании Kafka AdminClient",
		EN: "Kafka AdminClient creation error",
		CN: "创建 Kafka 管理客户端时出错",
	}

	MsgKafkaAdminMetadataError = Message{
		RU: "Ошибка получения метаданных Kafka",
		EN: "Failed to fetch Kafka metadata",
		CN: "获取 Kafka 元数据失败",
	}

	MsgKafkaAdminTopicMarkedForDeletion = Message{
		RU: "Топик помечен для удаления",
		EN: "Topic marked for deletion",
		CN: "主题已标记为删除",
	}

	MsgKafkaAdminDeleteError = Message{
		RU: "Ошибка при удалении топиков Kafka",
		EN: "Error while deleting Kafka topics",
		CN: "删除 Kafka 主题时出错",
	}

	MsgKafkaAdminDeleteSuccess = Message{
		RU: "Топик успешно удалён",
		EN: "Topic successfully deleted",
		CN: "主题删除成功",
	}

	MsgKafkaAdminNoTopicsToDelete = Message{
		RU: "Нет топиков для удаления",
		EN: "No topics to delete",
		CN: "没有需要删除的主题",
	}

	// MsgKafkaAdminClientCreateError = Message{RU: "Ошибка при создании Kafka AdminClient", EN: "Kafka AdminClient creation error", CN: "创建 Kafka 管理客户端时出错"}
	// MsgKafkaAdminMetadataError     = Message{RU: "Ошибка получения метаданных Kafka", EN: "Failed to fetch Kafka metadata", CN: "获取 Kafka 元数据失败"}

	MsgKafkaAdminTopicMarkedForCreation = Message{RU: "Топик помечен для создания", EN: "Topic marked for creation", CN: "主题已标记为创建"}

	MsgKafkaAdminCreateError   = Message{RU: "Ошибка при создании топика", EN: "Kafka topic creation error", CN: "Kafka 主题创建错误"}
	MsgKafkaAdminCreateWarning = Message{RU: "Предупреждение при создании топика", EN: "Kafka topic creation warning", CN: "Kafka 主题创建警告"}
	MsgKafkaAdminCreateSuccess = Message{RU: "Топик создан успешно", EN: "Kafka topic created successfully", CN: "Kafka 主题创建成功"}

	MsgKafkaAdminConfigReadError = Message{RU: "Ошибка чтения конфигурации топика", EN: "Failed to read topic config", CN: "读取主题配置失败"}
	MsgKafkaAdminUpdateNeeded    = Message{RU: "Требуется изменение cleanup.policy", EN: "cleanup.policy needs update", CN: "需要更新 cleanup.policy"}

	MsgKafkaAdminUpdateError   = Message{RU: "Ошибка при обновлении конфигурации топика", EN: "Kafka topic config update error", CN: "Kafka 主题配置更新错误"}
	MsgKafkaAdminUpdateWarning = Message{RU: "Предупреждение при обновлении конфигурации", EN: "Kafka config update warning", CN: "Kafka 配置更新警告"}
	MsgKafkaAdminUpdateSuccess = Message{RU: "Конфигурация топика успешно обновлена", EN: "Kafka config updated successfully", CN: "Kafka 配置更新成功"}

	// SMTP service messages
	MsgSMTPEventReceived = Message{RU: "Событие получено", EN: "event received", CN: "收到事件"}
	MsgSMTPWorkerError   = Message{RU: "Ошибка worker", EN: "worker error", CN: "工作器错误"}
	MsgEmailSendFailed   = Message{RU: "Ошибка при отправке письма", EN: "email send error", CN: "发送邮件错误"}

	// Kafka engine messages
	MsgKafkaConsumerCreateError   = Message{RU: "Kafka ошибка создания консюмера", EN: "Kafka consumer creation error", CN: "Kafka消费者创建错误"}
	MsgKafkaMetadataError         = Message{RU: "Ошибка получения метаданных", EN: "metadata fetch error", CN: "获取元数据错误"}
	MsgKafkaTopicFound            = Message{RU: "Найден топик", EN: "topic found", CN: "找到主题"}
	MsgKafkaNoPatternTopics       = Message{RU: "Нет подходящих топиков", EN: "no matching topics", CN: "没有匹配的主题"}
	MsgKafkaSubscribeError        = Message{RU: "Неудачная подписка", EN: "subscription failed", CN: "订阅失败"}
	MsgKafkaConsumerCreated       = Message{RU: "Kafka consumer создан и подписан", EN: "Kafka consumer ready", CN: "Kafka消费者已创建并订阅"}
	MsgKafkaAssignError           = Message{RU: "Неудачная попытка назначений", EN: "assignment error", CN: "分配错误"}
	MsgKafkaPartitionsNotAssigned = Message{RU: "Разделы не назначены", EN: "partitions not assigned", CN: "分区未分配"}
	MsgKafkaPartitionAssigned     = Message{RU: "Назначен раздел", EN: "partition assigned", CN: "分配分区"}
	MsgKafkaPartitionRevoked      = Message{RU: "Раздел отозван", EN: "partition revoked", CN: "分区已撤销"}

	// Debezium topic retention and consumer lag
	MsgKafkaTopicRetentionSet    = Message{RU: "Срок хранения топика приведён к конфигу", EN: "topic retention set from config", CN: "已按配置设置主题保留时间"}
	MsgKafkaTopicRetentionFailed = Message{RU: "Не удалось изменить срок хранения топика, повтор позже", EN: "failed to set topic retention, will retry", CN: "设置主题保留时间失败，稍后重试"}
	MsgKafkaTopicNotCreatedYet   = Message{RU: "Топика ещё нет: Connect создаст его со сроком хранения из конфига", EN: "topic does not exist yet: Connect will create it with the configured retention", CN: "主题尚不存在：Connect 将按配置的保留时间创建"}
	MsgCDCLagSummary             = Message{RU: "Отставание CDC", EN: "CDC lag", CN: "CDC 延迟"}
	MsgCDCLagHigh                = Message{RU: "Отставание CDC приближается к сроку хранения топика: события могут быть удалены непрочитанными", EN: "CDC lag is approaching topic retention: events may be deleted unread", CN: "CDC 延迟接近主题保留时间：事件可能在未读取前被删除"}
	MsgCDCLagFailed              = Message{RU: "Не удалось рассчитать отставание CDC", EN: "failed to compute CDC lag", CN: "无法计算 CDC 延迟"}

	// WAL watcher and Debezium heartbeat
	MsgWALSummary           = Message{RU: "Слоты репликации", EN: "replication slots", CN: "复制槽"}
	MsgWALSlotMissing       = Message{RU: "Слот репликации источника не найден: коннектор не создан или слот удалён", EN: "replication slot of the source not found: connector not created or slot dropped", CN: "未找到源的复制槽：连接器未创建或复制槽已删除"}
	MsgWALSlotInactive      = Message{RU: "Слот репликации неактивен: Debezium его не читает, WAL копится", EN: "replication slot is inactive: Debezium is not reading it, WAL accumulates", CN: "复制槽未激活：Debezium 未读取，WAL 在累积"}
	MsgWALSlotRetained      = Message{RU: "Слот репликации держит много WAL", EN: "replication slot retains too much WAL", CN: "复制槽保留的 WAL 过多"}
	MsgWALSlotUnreserved    = Message{RU: "Слот репликации вот-вот будет потерян: удерживаемый WAL превысил max_slot_wal_keep_size", EN: "replication slot is about to be lost: retained WAL exceeds max_slot_wal_keep_size", CN: "复制槽即将丢失：保留的 WAL 超过 max_slot_wal_keep_size"}
	MsgWALSlotLost          = Message{RU: "Слот репликации потерян: CDC по источнику остановлен, нужно восстановление (docs/CDC_OPERATIONS.md)", EN: "replication slot is lost: CDC for the source has stopped, recovery required (docs/CDC_OPERATIONS.md)", CN: "复制槽已丢失：该源的 CDC 已停止，需要恢复（docs/CDC_OPERATIONS.md）"}
	MsgWALSlotOrphan        = Message{RU: "Слот репликации не относится к источникам из конфига: если источник удалён, удалите слот (docs/CDC_OPERATIONS.md)", EN: "replication slot does not belong to any configured source: if the source was removed, drop the slot (docs/CDC_OPERATIONS.md)", CN: "复制槽不属于任何已配置的源：如果源已移除，请删除该复制槽（docs/CDC_OPERATIONS.md）"}
	MsgWALNoSlotLimit       = Message{RU: "У Postgres не задан max_slot_wal_keep_size: если Debezium остановится, WAL может заполнить диск", EN: "max_slot_wal_keep_size is not set: if Debezium stops, WAL may fill the disk", CN: "未设置 max_slot_wal_keep_size：如果 Debezium 停止，WAL 可能占满磁盘"}
	MsgWALSlotLimitUnknown  = Message{RU: "Не удалось проверить лимит WAL для слотов (нужен Postgres 13+)", EN: "failed to check the slot WAL limit (Postgres 13+ required)", CN: "无法检查复制槽 WAL 限制（需要 Postgres 13+）"}
	MsgWALCheckFailed       = Message{RU: "Не удалось проверить слоты репликации источника", EN: "failed to check replication slots of the source", CN: "无法检查源的复制槽"}
	MsgHeartbeatTableFailed = Message{RU: "Не удалось создать heartbeat-таблицу, коннектор зарегистрирован без неё", EN: "failed to create the heartbeat table, connector registered without it", CN: "无法创建 heartbeat 表，连接器在没有它的情况下注册"}
)
