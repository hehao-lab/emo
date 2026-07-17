CREATE TABLE IF NOT EXISTS users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '用户ID',
  username VARCHAR(64) NOT NULL UNIQUE COMMENT '登录用户名',
  password_hash VARCHAR(255) NOT NULL COMMENT '密码哈希值',
  phone VARCHAR(20) UNIQUE COMMENT '手机号',
  email VARCHAR(128) COMMENT '邮箱',
  avatar VARCHAR(512) DEFAULT '' COMMENT '头像地址',
  roles JSON NOT NULL COMMENT '角色列表JSON',
  status TINYINT NOT NULL DEFAULT 1 COMMENT '账号状态 1正常 2冻结 3注销',
  last_login_at DATETIME NULL COMMENT '最后登录时间',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  deleted_at DATETIME NULL COMMENT '软删除时间',
  INDEX idx_users_deleted_at (deleted_at),
  UNIQUE KEY idx_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户账号表';

CREATE TABLE IF NOT EXISTS user_profiles (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '资料ID',
  user_id BIGINT NOT NULL UNIQUE COMMENT '用户ID',
  nickname VARCHAR(64) DEFAULT '' COMMENT '昵称',
  avatar_url VARCHAR(512) DEFAULT '' COMMENT '头像地址',
  gender VARCHAR(16) DEFAULT '' COMMENT '性别',
  birthday VARCHAR(16) DEFAULT '' COMMENT '生日',
  bio VARCHAR(512) DEFAULT '' COMMENT '个人简介',
  location VARCHAR(128) DEFAULT '' COMMENT '所在地区',
  occupation VARCHAR(128) DEFAULT '' COMMENT '职业',
  industry VARCHAR(128) DEFAULT '' COMMENT '行业',
  language VARCHAR(32) DEFAULT 'zh-CN' COMMENT '语言偏好',
  timezone VARCHAR(64) DEFAULT '' COMMENT '时区',
  extra JSON NULL COMMENT '扩展资料JSON',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  deleted_at DATETIME NULL COMMENT '软删除时间',
  INDEX idx_user_profiles_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户资料表';

CREATE TABLE IF NOT EXISTS personal_profiles (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '个人信息ID',
  user_id BIGINT NOT NULL UNIQUE COMMENT '用户ID',
  age INT DEFAULT 0 COMMENT '年龄',
  gender VARCHAR(32) DEFAULT '' COMMENT '性别',
  mbti VARCHAR(16) DEFAULT '' COMMENT 'MBTI人格',
  relationship_status VARCHAR(128) DEFAULT '' COMMENT '关系说明',
  personality_summary TEXT NULL COMMENT '性格评价',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  deleted_at DATETIME NULL COMMENT '软删除时间',
  INDEX idx_personal_profiles_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='个人信息表';

CREATE TABLE IF NOT EXISTS target_profiles (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '目标信息ID',
  user_id BIGINT NOT NULL COMMENT '用户ID',
  personal_profile_id BIGINT NOT NULL COMMENT '个人信息ID',
  name VARCHAR(128) DEFAULT '' COMMENT '对方称呼',
  age INT DEFAULT 0 COMMENT '对方年龄',
  gender VARCHAR(32) DEFAULT '' COMMENT '对方性别',
  mbti VARCHAR(16) DEFAULT '' COMMENT 'MBTI人格',
  current_relationship VARCHAR(128) DEFAULT '' COMMENT '当前关系',
  interaction_frequency VARCHAR(128) DEFAULT '' COMMENT '互动频率',
  relationship_goal VARCHAR(256) DEFAULT '' COMMENT '关系目标',
  personality_traits TEXT NULL COMMENT '对方性格与相处特点',
  recent_interaction TEXT NULL COMMENT '最近一次关键互动',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  deleted_at DATETIME NULL COMMENT '软删除时间',
  INDEX idx_target_profiles_user_id (user_id),
  INDEX idx_target_profiles_personal_profile_id (personal_profile_id),
  INDEX idx_target_profiles_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='目标信息表';

CREATE TABLE IF NOT EXISTS important_records (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '重要记录ID',
  user_id BIGINT NOT NULL COMMENT '用户ID',
  personal_profile_id BIGINT NOT NULL COMMENT '个人信息ID',
  target_profile_id BIGINT NOT NULL COMMENT '目标信息ID',
  title VARCHAR(160) NOT NULL COMMENT '标题',
  record_time VARCHAR(32) DEFAULT '' COMMENT '记录时间',
  event_description TEXT NULL COMMENT '事件描述',
  resolution TEXT NULL COMMENT '矛盾解决方式',
  concern_point TEXT NULL COMMENT '在意的点',
  satisfaction VARCHAR(32) DEFAULT '' COMMENT '满意度',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  deleted_at DATETIME NULL COMMENT '软删除时间',
  INDEX idx_important_records_user_id (user_id),
  INDEX idx_important_records_personal_profile_id (personal_profile_id),
  INDEX idx_important_records_target_profile_id (target_profile_id),
  INDEX idx_important_records_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='重要记录表';

CREATE TABLE IF NOT EXISTS auth_refresh_tokens (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '刷新令牌记录ID',
  user_id BIGINT NOT NULL COMMENT '用户ID',
  token_id VARCHAR(64) NOT NULL UNIQUE COMMENT '刷新令牌唯一ID',
  token_hash VARCHAR(255) NOT NULL UNIQUE COMMENT '刷新令牌哈希值',
  device_id VARCHAR(128) DEFAULT '' COMMENT '设备ID',
  device_name VARCHAR(128) DEFAULT '' COMMENT '设备名称',
  ip VARCHAR(64) DEFAULT '' COMMENT '登录IP',
  user_agent VARCHAR(512) DEFAULT '' COMMENT '客户端User-Agent',
  expires_at DATETIME NOT NULL COMMENT '过期时间',
  revoked_at DATETIME NULL COMMENT '撤销时间',
  revoke_reason VARCHAR(128) DEFAULT '' COMMENT '撤销原因',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  deleted_at DATETIME NULL COMMENT '软删除时间',
  INDEX idx_auth_tokens_user_id (user_id),
  INDEX idx_auth_tokens_expires_at (expires_at),
  INDEX idx_auth_tokens_revoked_at (revoked_at),
  INDEX idx_auth_tokens_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='刷新令牌表';

CREATE TABLE IF NOT EXISTS login_logs (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '登录日志ID',
  user_id BIGINT NULL COMMENT '用户ID',
  username VARCHAR(64) COMMENT '登录用户名',
  login_type VARCHAR(32) DEFAULT 'password' COMMENT '登录类型',
  success BOOLEAN NOT NULL COMMENT '是否登录成功',
  fail_reason VARCHAR(128) DEFAULT '' COMMENT '失败原因',
  ip VARCHAR(64) DEFAULT '' COMMENT '登录IP',
  user_agent VARCHAR(512) DEFAULT '' COMMENT '客户端User-Agent',
  device_id VARCHAR(128) DEFAULT '' COMMENT '设备ID',
  location VARCHAR(128) DEFAULT '' COMMENT '登录地理位置',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  INDEX idx_login_logs_user_id (user_id),
  INDEX idx_login_logs_username (username),
  INDEX idx_login_logs_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='登录日志表';

CREATE TABLE IF NOT EXISTS security_events (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '安全事件ID',
  user_id BIGINT NOT NULL COMMENT '用户ID',
  event_type VARCHAR(64) NOT NULL COMMENT '事件类型',
  risk_level VARCHAR(16) DEFAULT 'low' COMMENT '风险等级',
  ip VARCHAR(64) DEFAULT '' COMMENT '操作IP',
  user_agent VARCHAR(512) DEFAULT '' COMMENT '客户端User-Agent',
  metadata_json JSON NULL COMMENT '事件扩展信息JSON',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  INDEX idx_security_events_user_id (user_id),
  INDEX idx_security_events_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='安全事件表';

CREATE TABLE IF NOT EXISTS mood_diaries (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '心情日记ID',
  user_id BIGINT NOT NULL COMMENT '用户ID',
  title VARCHAR(128) DEFAULT '' COMMENT '日记标题',
  content TEXT NOT NULL COMMENT '日记正文',
  mood VARCHAR(32) DEFAULT '' COMMENT '心情类型',
  mood_score TINYINT DEFAULT 0 COMMENT '心情分数 1到10',
  weather VARCHAR(32) DEFAULT '' COMMENT '天气',
  location VARCHAR(128) DEFAULT '' COMMENT '记录地点',
  occurred_on DATE NOT NULL COMMENT '日记发生日期',
  visibility VARCHAR(16) DEFAULT 'private' COMMENT '可见性',
  analysis_id BIGINT DEFAULT 0 COMMENT '关联情绪分析ID',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  deleted_at DATETIME NULL COMMENT '软删除时间',
  INDEX idx_mood_diaries_user_day (user_id, occurred_on),
  INDEX idx_mood_diaries_mood (mood),
  INDEX idx_mood_diaries_analysis_id (analysis_id),
  INDEX idx_mood_diaries_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='心情日记表';

CREATE TABLE IF NOT EXISTS mood_tags (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '心情标签ID',
  user_id BIGINT NOT NULL DEFAULT 0 COMMENT '用户ID 0表示系统标签',
  name VARCHAR(32) NOT NULL COMMENT '标签名称',
  color VARCHAR(16) DEFAULT '' COMMENT '标签颜色',
  icon VARCHAR(64) DEFAULT '' COMMENT '标签图标',
  sort INT DEFAULT 0 COMMENT '排序值',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  deleted_at DATETIME NULL COMMENT '软删除时间',
  INDEX idx_mood_tags_user_id (user_id),
  INDEX idx_mood_tags_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='心情标签表';

CREATE TABLE IF NOT EXISTS mood_diary_tags (
  diary_id BIGINT NOT NULL COMMENT '心情日记ID',
  tag_id BIGINT NOT NULL COMMENT '心情标签ID',
  PRIMARY KEY (diary_id, tag_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='心情日记标签关系表';

CREATE TABLE IF NOT EXISTS mood_diary_attachments (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '日记附件ID',
  diary_id BIGINT NOT NULL COMMENT '心情日记ID',
  file_id BIGINT DEFAULT 0 COMMENT '文件资源ID',
  url VARCHAR(1024) NOT NULL COMMENT '附件访问地址',
  sort INT DEFAULT 0 COMMENT '排序值',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  INDEX idx_mood_diary_attachments_diary_id (diary_id),
  INDEX idx_mood_diary_attachments_file_id (file_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='心情日记附件表';

CREATE TABLE IF NOT EXISTS chat_sessions (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '聊天会话ID',
  user_id BIGINT NOT NULL COMMENT '用户ID',
  title VARCHAR(128) DEFAULT '' COMMENT '会话标题',
  scenario VARCHAR(64) DEFAULT 'emotional_support' COMMENT '咨询场景',
  status VARCHAR(16) DEFAULT 'active' COMMENT '会话状态',
  summary TEXT NULL COMMENT '会话摘要',
  upstream_conversation_id VARCHAR(128) DEFAULT '' COMMENT '上游AI会话ID',
  last_message_at DATETIME NULL COMMENT '最后消息时间',
  message_count INT DEFAULT 0 COMMENT '消息数量',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  deleted_at DATETIME NULL COMMENT '软删除时间',
  INDEX idx_chat_sessions_user_last (user_id, last_message_at),
  INDEX idx_chat_sessions_upstream_conversation_id (upstream_conversation_id),
  INDEX idx_chat_sessions_status (status),
  INDEX idx_chat_sessions_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='聊天会话表';

CREATE TABLE IF NOT EXISTS chat_messages (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '聊天消息ID',
  session_id BIGINT NOT NULL COMMENT '聊天会话ID',
  user_id BIGINT NOT NULL COMMENT '用户ID',
  role VARCHAR(16) NOT NULL COMMENT '消息角色 user assistant system tool',
  content TEXT NOT NULL COMMENT '消息内容',
  content_type VARCHAR(32) DEFAULT 'text' COMMENT '消息内容类型',
  model VARCHAR(64) DEFAULT '' COMMENT 'AI模型名称',
  prompt_tokens INT DEFAULT 0 COMMENT '提示词token数',
  completion_tokens INT DEFAULT 0 COMMENT '回复token数',
  total_tokens INT DEFAULT 0 COMMENT '总token数',
  latency_ms INT DEFAULT 0 COMMENT 'AI回复耗时毫秒',
  emotion_snapshot_json JSON NULL COMMENT '消息情绪快照JSON',
  safety_result_json JSON NULL COMMENT '安全检测结果JSON',
  status VARCHAR(16) DEFAULT 'success' COMMENT '消息状态',
  error_message VARCHAR(512) DEFAULT '' COMMENT '错误信息',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  INDEX idx_chat_messages_session_created (session_id, created_at),
  INDEX idx_chat_messages_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='聊天消息表';

CREATE TABLE IF NOT EXISTS chat_context_summaries (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '会话摘要ID',
  session_id BIGINT NOT NULL COMMENT '聊天会话ID',
  summary TEXT NOT NULL COMMENT '摘要内容',
  message_start_id BIGINT DEFAULT 0 COMMENT '摘要起始消息ID',
  message_end_id BIGINT DEFAULT 0 COMMENT '摘要结束消息ID',
  model VARCHAR(64) DEFAULT '' COMMENT '摘要模型名称',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  INDEX idx_chat_context_summaries_session_id (session_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='聊天上下文摘要表';

CREATE TABLE IF NOT EXISTS chat_feedback (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '消息反馈ID',
  user_id BIGINT NOT NULL COMMENT '用户ID',
  session_id BIGINT NOT NULL COMMENT '聊天会话ID',
  message_id BIGINT NOT NULL COMMENT '被反馈的消息ID',
  rating TINYINT DEFAULT 0 COMMENT '评分',
  feedback_type VARCHAR(32) DEFAULT '' COMMENT '反馈类型',
  content VARCHAR(512) DEFAULT '' COMMENT '反馈内容',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  INDEX idx_chat_feedback_user_id (user_id),
  INDEX idx_chat_feedback_session_id (session_id),
  INDEX idx_chat_feedback_message_id (message_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='聊天反馈表';

CREATE TABLE IF NOT EXISTS emotion_analyses (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '情绪分析ID',
  user_id BIGINT NOT NULL COMMENT '用户ID',
  source_type VARCHAR(32) NOT NULL COMMENT '分析来源类型 diary chat_message manual',
  source_id BIGINT DEFAULT 0 COMMENT '来源数据ID',
  primary_emotion VARCHAR(32) DEFAULT '' COMMENT '主导情绪',
  sentiment VARCHAR(16) DEFAULT 'neutral' COMMENT '情感倾向 positive neutral negative',
  sentiment_score DECIMAL(5,4) DEFAULT 0 COMMENT '情感分数 -1到1',
  stress_score TINYINT DEFAULT 0 COMMENT '压力分数 0到100',
  anxiety_score TINYINT DEFAULT 0 COMMENT '焦虑分数 0到100',
  depression_risk_score TINYINT DEFAULT 0 COMMENT '抑郁风险分数 0到100',
  energy_score TINYINT DEFAULT 0 COMMENT '能量分数 0到100',
  confidence DECIMAL(5,4) DEFAULT 0 COMMENT '分析置信度',
  summary TEXT NULL COMMENT '分析摘要',
  advice TEXT NULL COMMENT '建议内容',
  risk_level VARCHAR(16) DEFAULT 'low' COMMENT '风险等级 low medium high crisis',
  model VARCHAR(64) DEFAULT '' COMMENT '分析模型名称',
  raw_result_json JSON NULL COMMENT '原始分析结果JSON',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  INDEX idx_emotion_analyses_user_created (user_id, created_at),
  INDEX idx_emotion_analyses_source_type (source_type),
  INDEX idx_emotion_analyses_source_id (source_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='情绪分析表';

CREATE TABLE IF NOT EXISTS emotion_dimension_scores (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '情绪维度分数ID',
  analysis_id BIGINT NOT NULL COMMENT '情绪分析ID',
  dimension VARCHAR(64) NOT NULL COMMENT '情绪维度名称',
  score DECIMAL(5,4) NOT NULL COMMENT '维度分数 0到1',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  INDEX idx_emotion_dimension_scores_analysis_id (analysis_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='情绪维度分数表';

CREATE TABLE IF NOT EXISTS emotion_daily_stats (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '每日情绪统计ID',
  user_id BIGINT NOT NULL COMMENT '用户ID',
  stat_date DATE NOT NULL COMMENT '统计日期',
  diary_count INT DEFAULT 0 COMMENT '当日心情日记数量',
  chat_count INT DEFAULT 0 COMMENT '当日聊天消息数量',
  avg_mood_score DECIMAL(5,2) DEFAULT 0 COMMENT '平均心情分数',
  avg_sentiment_score DECIMAL(5,4) DEFAULT 0 COMMENT '平均情感分数',
  dominant_emotion VARCHAR(32) DEFAULT '' COMMENT '当日主导情绪',
  high_risk_count BIGINT DEFAULT 0 COMMENT '高风险分析次数',
  dimension_summary JSON NULL COMMENT '情绪维度汇总JSON',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  UNIQUE KEY uk_emotion_daily_stats_user_date (user_id, stat_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='每日情绪统计表';

CREATE TABLE IF NOT EXISTS system_configs (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '系统配置ID',
  config_key VARCHAR(128) NOT NULL UNIQUE COMMENT '配置键',
  config_value JSON NOT NULL COMMENT '配置值JSON',
  description VARCHAR(255) DEFAULT '' COMMENT '配置说明',
  is_public BOOLEAN DEFAULT FALSE COMMENT '是否公开给前端读取',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统配置表';

CREATE TABLE IF NOT EXISTS app_versions (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '应用版本ID',
  platform VARCHAR(16) NOT NULL COMMENT '平台 ios android web',
  version VARCHAR(32) NOT NULL COMMENT '版本号',
  build_no INT NOT NULL COMMENT '构建号',
  force_update BOOLEAN DEFAULT FALSE COMMENT '是否强制更新',
  download_url VARCHAR(1024) DEFAULT '' COMMENT '下载地址',
  changelog TEXT NULL COMMENT '更新说明',
  min_supported_version VARCHAR(32) DEFAULT '' COMMENT '最低支持版本',
  published_at DATETIME NULL COMMENT '发布时间',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  INDEX idx_app_versions_platform (platform),
  INDEX idx_app_versions_build_no (build_no),
  INDEX idx_app_versions_published_at (published_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='应用版本表';

CREATE TABLE IF NOT EXISTS system_announcements (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '系统公告ID',
  title VARCHAR(128) NOT NULL COMMENT '公告标题',
  content TEXT NOT NULL COMMENT '公告内容',
  target_platform VARCHAR(32) DEFAULT 'all' COMMENT '目标平台',
  start_at DATETIME NULL COMMENT '生效开始时间',
  end_at DATETIME NULL COMMENT '生效结束时间',
  status TINYINT DEFAULT 1 COMMENT '公告状态 1启用 0停用',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  INDEX idx_system_announcements_platform (target_platform),
  INDEX idx_system_announcements_start_at (start_at),
  INDEX idx_system_announcements_end_at (end_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统公告表';

CREATE TABLE IF NOT EXISTS file_assets (
  id BIGINT PRIMARY KEY AUTO_INCREMENT COMMENT '文件资源ID',
  owner_user_id BIGINT DEFAULT 0 COMMENT '所属用户ID',
  biz_type VARCHAR(32) DEFAULT '' COMMENT '业务类型 avatar diary system',
  storage_provider VARCHAR(32) DEFAULT 'local' COMMENT '存储服务商',
  bucket VARCHAR(128) DEFAULT '' COMMENT '存储桶',
  object_key VARCHAR(512) NOT NULL COMMENT '对象存储Key',
  url VARCHAR(1024) NOT NULL COMMENT '文件访问地址',
  mime_type VARCHAR(128) DEFAULT '' COMMENT '文件MIME类型',
  size_bytes BIGINT DEFAULT 0 COMMENT '文件大小字节数',
  checksum VARCHAR(128) DEFAULT '' COMMENT '文件校验值',
  status TINYINT DEFAULT 1 COMMENT '文件状态',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  deleted_at DATETIME NULL COMMENT '软删除时间',
  INDEX idx_file_assets_owner_user_id (owner_user_id),
  INDEX idx_file_assets_biz_type (biz_type),
  INDEX idx_file_assets_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文件资源表';
