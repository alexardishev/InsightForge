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
)
