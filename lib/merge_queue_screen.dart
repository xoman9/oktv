import 'dart:async';
import 'dart:io';

import 'package:flutter/material.dart';

import 'core_bridge.dart';
import 'local_store.dart';
import 'media_library.dart';
import 'models.dart';
import 'remote_widgets.dart';
import 'widgets.dart';

class MergeQueueScreen extends StatefulWidget {
  const MergeQueueScreen({
    super.key,
    required this.library,
    required this.repository,
    required this.store,
    this.addOnOpen = false,
  });
  final MediaLibrary library;
  final AppRepository repository;
  final LocalStore store;
  final bool addOnOpen;
  @override
  State<MergeQueueScreen> createState() => _MergeQueueScreenState();
}

class _MergeQueueScreenState extends State<MergeQueueScreen> {
  bool _loading = true, _adding = false;
  String? _error;
  late final int _epoch = widget.store.profileEpoch;
  bool get _allowed =>
      widget.store.canDownload && widget.store.profileEpoch == _epoch;
  @override
  void initState() {
    super.initState();
    ensureTelevisionFocus(context);
    _load();
  }

  Future<void> _load() async {
    try {
      await widget.library.merges.load();
      if (mounted && _allowed && widget.addOnOpen) await _add();
    } catch (error) {
      if (mounted) setState(() => _error = error.toString());
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _add() async {
    if (_adding || !_allowed) return;
    setState(() => _adding = true);
    try {
      final jobs = await widget.repository.downloads();
      if (!mounted || !_allowed) return;
      final groups = <String, List<DownloadJob>>{};
      for (final job in jobs.where((job) => job.completed)) {
        (groups[job.drama.id] ??= []).add(job);
      }
      final rows = groups.entries.toList();
      final errors = <String, String>{};
      for (final row in rows) {
        try {
          continuousMergeJobs(row.value);
        } catch (error) {
          errors[row.key] = error.toString();
        }
      }
      final selected = <String>{};
      var cleanup = true;
      var skipVerification = true;
      final accepted = await showDialog<bool>(
        context: context,
        builder: (context) => StatefulBuilder(
          builder: (context, change) => AlertDialog(
            title: const Text('选择要合并的剧'),
            content: SizedBox(
              width: 520,
              height: MediaQuery.sizeOf(context).height * .6,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  const Text('可從 1 集開始合併已下載的連續分集，不限制合併集數；中斷後可繼續。'),
                  const SizedBox(height: 8),
                  Expanded(
                    child: rows.isEmpty
                        ? const Center(child: Text('請先下載至少 1 集影片'))
                        : ListView.builder(
                            itemCount: rows.length,
                            itemBuilder: (_, index) {
                              final row = rows[index];
                              return CheckboxListTile(
                                contentPadding: EdgeInsets.zero,
                                value: selected.contains(row.key),
                                title: Text(row.value.first.drama.title),
                                subtitle: Text(
                                  errors[row.key] ??
                                      '已下载 ${row.value.length} 集 · 第 ${row.value.map((job) => job.episode.number).reduce((a, b) => a < b ? a : b)}–${row.value.map((job) => job.episode.number).reduce((a, b) => a > b ? a : b)} 集',
                                ),
                                onChanged: errors.containsKey(row.key)
                                    ? null
                                    : (value) => change(() {
                                        if (value == true) {
                                          selected.add(row.key);
                                        } else {
                                          selected.remove(row.key);
                                        }
                                      }),
                              );
                            },
                          ),
                  ),
                  CheckboxListTile(
                    contentPadding: EdgeInsets.zero,
                    value: skipVerification,
                    title: const Text('免驗證快速合併'),
                    onChanged: (value) => change(() {
                      skipVerification = value == true;
                    }),
                  ),
                  CheckboxListTile(
                    contentPadding: EdgeInsets.zero,
                    value: cleanup,
                    title: const Text('完成后删除原分集'),
                    subtitle: const Text('合併檔成功儲存後清理原分集；失敗時保留原分集。'),
                    onChanged: (value) => change(() => cleanup = value == true),
                  ),
                ],
              ),
            ),
            actions: [
              TextButton(
                onPressed: () => Navigator.pop(context, false),
                child: const Text('取消'),
              ),
              FilledButton(
                onPressed: selected.isEmpty
                    ? null
                    : () => Navigator.pop(context, true),
                child: Text('加入 ${selected.length} 部'),
              ),
            ],
          ),
        ),
      );
      if (accepted != true || !mounted || !_allowed) return;
      if (skipVerification && Platform.isAndroid && !(await ensureSharedMergeStorage())) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('請在系統設定允許所有檔案存取權限，返回後重新加入快速合併')),
          );
        }
        return;
      }
      final count = await widget.library.merges.enqueue([
        for (final id in selected) groups[id]!,
      ], cleanup: cleanup, skipVerification: skipVerification);
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('已加入 $count 部，已有合并任务自动跳过')));
      }
    } catch (error) {
      if (mounted) setState(() => _error = error.toString());
    } finally {
      if (mounted) setState(() => _adding = false);
    }
  }

  Future<void> _control(MergeQueueJob job, String action) async {
    if (!_allowed) return;
    try {
      await widget.library.merges.control(job.id, action);
    } catch (error) {
      if (mounted) setState(() => _error = error.toString());
    }
  }

  @override
  Widget build(BuildContext context) => AnimatedBuilder(
    animation: widget.library,
    builder: (context, _) {
      final jobs = widget.library.merges.items;
      return Scaffold(
        appBar: AppBar(
          title: const Text('合并队列'),
          actions: [
            IconButton(
              tooltip: '添加多部剧',
              onPressed: _adding || !_allowed ? null : _add,
              icon: const Icon(Icons.add_rounded),
            ),
          ],
        ),
        body: !_allowed
            ? const StatusPanel(title: '当前用户已切换', message: '请返回后重新打开合并队列。')
            : _loading
            ? const Center(child: CircularProgressIndicator())
            : Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  if (_adding) const LinearProgressIndicator(),
                  if (_error != null || widget.library.merges.error.isNotEmpty)
                    Padding(
                      padding: const EdgeInsets.all(16),
                      child: Text(_error ?? widget.library.merges.error),
                    ),
                  Expanded(
                    child: jobs.isEmpty
                        ? StatusPanel(
                            title: '还没有合并任务',
                            message: '选择多部剧加入队列，依次合并已下载的连续分集。',
                            secondaryAction: FilledButton.icon(
                              onPressed: _adding ? null : _add,
                              icon: const Icon(Icons.add_rounded),
                              label: const Text('添加剧集'),
                            ),
                          )
                        : ListView.builder(
                            padding: const EdgeInsets.all(12),
                            itemCount: jobs.length,
                            itemBuilder: (_, index) {
                              final job = jobs[index];
                              final processing =
                                  job.state == 'running' ||
                                  job.state == 'cleanup';
                              return Card(
                                child: Padding(
                                  padding: const EdgeInsets.all(16),
                                  child: Column(
                                    crossAxisAlignment:
                                        CrossAxisAlignment.stretch,
                                    children: [
                                      Text(
                                        job.drama.title,
                                        style: Theme.of(
                                          context,
                                        ).textTheme.titleMedium,
                                      ),
                                      const SizedBox(height: 6),
                                      Text(
                                        '第 ${job.episodes.first}–${job.episodes.last} 集 · ${job.label}${job.cleanup ? ' · 完成后清理分集' : ''}',
                                      ),
                                      if (processing) ...[
                                        const SizedBox(height: 10),
                                        LinearProgressIndicator(
                                          value: widget.library.progress > 0
                                              ? widget.library.progress
                                              : null,
                                        ),
                                        const SizedBox(height: 6),
                                        Text(widget.library.status),
                                      ],
                                      if (job.error.isNotEmpty)
                                        Padding(
                                          padding: const EdgeInsets.only(
                                            top: 8,
                                          ),
                                          child: Text(job.error),
                                        ),
                                      Wrap(
                                        spacing: 8,
                                        children: [
                                          if (job.active)
                                            TextButton.icon(
                                              onPressed: () =>
                                                  _control(job, 'pause'),
                                              icon: const Icon(
                                                Icons.pause_rounded,
                                              ),
                                              label: const Text('暂停'),
                                            ),
                                          if (job.state == 'paused' ||
                                              job.state == 'failed')
                                            TextButton.icon(
                                              onPressed: () =>
                                                  _control(job, 'resume'),
                                              icon: const Icon(
                                                Icons.play_arrow_rounded,
                                              ),
                                              label: Text(
                                                job.state == 'failed'
                                                    ? '重试'
                                                    : '继续',
                                              ),
                                            ),
                                          if (job.state != 'completed' &&
                                              job.state != 'cancelled')
                                            TextButton(
                                              onPressed: () =>
                                                  _control(job, 'cancel'),
                                              child: const Text('取消任务'),
                                            ),
                                          if (job.state == 'completed' ||
                                              job.state == 'cancelled')
                                            TextButton(
                                              onPressed: () =>
                                                  _control(job, 'forget'),
                                              child: const Text('清理记录'),
                                            ),
                                        ],
                                      ),
                                    ],
                                  ),
                                ),
                              );
                            },
                          ),
                  ),
                ],
              ),
      );
    },
  );
}
