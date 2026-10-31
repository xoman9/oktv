part of 'media_library.dart';

List<DownloadJob> continuousMergeJobs(List<DownloadJob> selected) {
  final jobs = List<DownloadJob>.of(selected)
    ..sort((a, b) => a.episode.number.compareTo(b.episode.number));
  if (jobs.isEmpty ||
      jobs.any((job) => !job.completed) ||
      jobs.map((job) => job.drama.id).toSet().length != 1 ||
      jobs.map((job) => job.episode.number).toSet().length != jobs.length) {
    throw AppFailure('請選擇同一部劇至少 1 集已下載的影片，集數不可重複');
  }
  final missing = <int>[];
  for (var i = 1; i < jobs.length; i++) {
    for (
      var number = jobs[i - 1].episode.number + 1;
      number < jobs[i].episode.number && missing.length < 12;
      number++
    ) {
      missing.add(number);
    }
  }
  if (missing.isNotEmpty) {
    throw AppFailure(
      '分集不连续，缺少第 ${missing.join('、')}${missing.length == 12 ? '…' : ''} 集，请补齐后合并',
    );
  }
  return jobs;
}

const _mergedStorageChannel = MethodChannel('duanju/device');

Future<bool> ensureSharedMergeStorage() async {
  if (!Platform.isAndroid) return true;
  final granted =
      await _mergedStorageChannel.invokeMethod<bool>('sharedStorageAccess') ??
      false;
  if (granted) return true;
  await _mergedStorageChannel.invokeMethod<bool>('requestSharedStorageAccess');
  return false;
}

// Publish directly into the selected download root; never create library/full.mp4.
// Reserve a unique name atomically, and keep the source until the copy is complete.
Future<File> _publishMergedMp4(
  File original,
  Drama drama,
  String directory,
  List<int> episodes,
) async {
  if (!path.isAbsolute(directory)) throw AppFailure('下載資料夾路徑無效');
  final base = Directory(directory);
  await base.create(recursive: true);
  final title = '${_safeName(drama.title)}(${episodes.first}-${episodes.last})';
  for (var n = 0; n < 10000; n++) {
    final name = n == 0 ? '$title.mp4' : '$title ($n).mp4';
    final dest = File(path.join(base.path, name));
    // File.create(exclusive: true) atomically reserves a name without
    // overwriting an existing merged video.
    try {
      await dest.create(exclusive: true);
    } on FileSystemException {
      if (await dest.exists()) continue;
      rethrow;
    }
    try {
      await original.openRead().pipe(dest.openWrite());
      if (await dest.length() != await original.length()) {
        throw AppFailure('合併成品儲存不完整，原分集已保留');
      }
      return dest;
    } catch (_) {
      if (await dest.exists()) await dest.delete();
      rethrow;
    }
  }
  throw AppFailure('合併成品同名檔案過多');
}

String _mergeDigest(Object value) =>
    sha256.convert(utf8.encode(jsonEncode(value))).toString();

extension MediaMergeOperations on MediaLibrary {
  Future<LocalMediaItem> merge(List<DownloadJob> selected) async {
    final jobs = continuousMergeJobs(selected);
    return _task(
      '准备合并',
      (temporary) => _mergeFiles(
        jobs,
        temporary,
        'merged-${DateTime.now().microsecondsSinceEpoch}',
      ),
      sources: {jobs.first.drama.source},
    );
  }

  Future<LocalMediaItem> _mergeQueued(MergeQueueJob task) => _task(
    '恢复合并进度',
    (temporary) async {
      final existing = _items
          .where((item) => item.id == task.outputId && item.merged)
          .firstOrNull;
      if (existing != null &&
          existing.drama.id == task.drama.id &&
          listEquals(existing.episodes, task.episodes) &&
          await File(fileFor(existing)).exists() &&
          await File(fileFor(existing)).length() == existing.bytes) {
        // Existing output is already in the selected download root.
        return existing;
      }
      final current = {
        for (final job in await repository.downloads()) job.id: job,
      };
      _check();
      final jobs = <DownloadJob>[];
      for (final id in task.jobIds) {
        final job = current[id];
        if (job == null ||
            !job.completed ||
            _sourceVersion(job) != task.versions[id]) {
          throw AppFailure('原分集已删除、重新下载或未完成，请重新建立合并任务；已保存的处理进度保留');
        }
        jobs.add(job);
      }
      return _mergeFiles(continuousMergeJobs(jobs), temporary, task.outputId);
    },
    workspace: task.id,
    sources: {task.drama.source},
  );

  Future<void> _decodeOutput(
    String file,
    double duration, {
    double base = .9,
    double weight = .09,
  }) async {
    _check();
    status = '完整解码校验成品';
    notifyListeners();
    await executor.run(
      [
        '-xerror',
        '-err_detect',
        'explode',
        '-i',
        file,
        '-map',
        '0:v:0',
        '-map',
        '0:a:0?',
        '-sn',
        '-dn',
        '-f',
        'null',
        '-',
      ],
      duration: duration,
      progress: (value) {
        progress = base + weight * value;
        notifyListeners();
      },
    );
    _check();
  }

  Future<LocalMediaItem> _mergeFiles(
    List<DownloadJob> jobs,
    Directory temporary,
    String id,
  ) async {
    final sources = <Object>[];
    for (final job in jobs) {
      _check();
      final plan = await repository.localPlayback(job.drama, job.episode);
      _check();
      if (plan == null || !plan.local) {
        throw AppFailure('第 ${job.episode.number} 集没有完整的本地文件');
      }
      final stat = await File(plan.url).stat();
      sources.add([
        job.id,
        _sourceVersion(job),
        job.episode.number,
        path.basename(plan.url),
        stat.size,
        stat.modified.microsecondsSinceEpoch,
        _mergeDigest(plan.decryptionKey),
      ]);
    }
    final fingerprint = _mergeDigest(['merge-v4-lowest', sources]);
    final checkpointFile = File(path.join(temporary.path, 'checkpoint.json'));
    var checkpoint = <String, dynamic>{
      'fingerprint': fingerprint,
      'prepared': <String, dynamic>{},
      'normalized': <String, dynamic>{},
    };
    if (await checkpointFile.exists()) {
      if (await checkpointFile.length() > 8 * 1024 * 1024) {
        throw AppFailure('合并恢复记录过大，已保留原文件');
      }
      checkpoint = Map<String, dynamic>.from(
        jsonDecode(await checkpointFile.readAsString()) as Map,
      );
      if (checkpoint['fingerprint'] != fingerprint) {
        throw AppFailure('原分集或合并参数已改变，请取消此任务后重新加入；原视频保留');
      }
    }
    final originals = Map<String, dynamic>.from(
      checkpoint['prepared'] as Map? ?? {},
    );
    final parts = Map<String, dynamic>.from(
      checkpoint['normalized'] as Map? ?? {},
    );
    checkpoint['prepared'] = originals;
    checkpoint['normalized'] = parts;

    Future<void> saveCheckpoint() async {
      _check();
      await _writeText(checkpointFile, jsonEncode(checkpoint));
    }

    Future<MediaProbe?> restored(
      Map<String, dynamic> records,
      String key,
      String file,
    ) async {
      final raw = records[key];
      if (raw is! Map) return null;
      final stat = await File(file).stat();
      if (stat.type != FileSystemEntityType.file ||
          stat.size != raw['bytes'] ||
          stat.modified.microsecondsSinceEpoch != raw['modified']) {
        return null;
      }
      try {
        final saved = MediaProbe(
          Map<String, dynamic>.from(raw['probe'] as Map),
        );
        return saved;
      } catch (_) {
        _check();
        return null;
      }
    }

    Future<void> record(
      Map<String, dynamic> records,
      String key,
      String file,
      MediaProbe probe,
    ) async {
      final stat = await File(file).stat();
      records[key] = {
        'bytes': stat.size,
        'modified': stat.modified.microsecondsSinceEpoch,
        'probe': probe.raw,
      };
      await saveCheckpoint();
    }

    final prepared = <String>[], probes = <MediaProbe>[];
    for (var i = 0; i < jobs.length; i++) {
      _check();
      final target = path.join(temporary.path, 'original-$i.mkv');
      status = '預備第 ${jobs[i].episode.number} 集';
      notifyListeners();
      var probe = await restored(originals, jobs[i].id, target);
      if (probe == null) {
        probe = await _prepare(
          jobs[i],
          target,
          .2 * i / jobs.length,
          .2 / jobs.length,
          skipVerification: true,
        );
        await record(originals, jobs[i].id, target, probe);
      }
      probes.add(probe);
      prepared.add(target);
    }
    final plan = MergePlan.create(probes);
    final planKey = _mergeDigest([
      plan.video.videoSignature,
      plan.audio?.audioSignature,
      plan.videoChanges,
      plan.audioChanges,
      plan.canonicalAac,
      plan.transportStream,
    ]);
    if (checkpoint['plan'] != planKey) {
      parts.clear();
      checkpoint['plan'] = planKey;
      await saveCheckpoint();
    }
    final normalized = <String>[];
    for (var i = 0; i < jobs.length; i++) {
      _check();
      status = plan.videoChanges[i]
          ? '第 ${jobs[i].episode.number} 集格式不符，轉檔中'
          : plan.audioChanges[i]
          ? '第 ${jobs[i].episode.number} 集格式不符，轉檔中'
          : '預備第 ${jobs[i].episode.number} 集';
      notifyListeners();
      final target = path.join(
        temporary.path,
        'part-$i.${plan.transportStream ? 'ts' : 'mkv'}',
      );
      var probe = await restored(parts, jobs[i].id, target);
      if (probe == null) {
        await executor.run(
          plan.normalizeArguments(prepared[i], target, probes[i], i),
          duration: probes[i].duration,
          progress: (value) {
            progress = .2 + .5 * (i + value) / jobs.length;
            notifyListeners();
          },
        );
        _check();
        probe = MediaProbe({
          'format': {'duration': '${probes[i].duration}'},
          'streams': [
            plan.video.video,
            if (plan.audio != null) plan.audio!.audio,
          ],
        });
        await record(parts, jobs[i].id, target, probe);
      }
      normalized.add(target);
    }
    _check();
    status = '合併第 ${jobs.first.episode.number} 集中';
    notifyListeners();
    final list = File(path.join(temporary.path, 'concat.txt'));
    await list.writeAsString(
      'ffconcat version 1.0\n${normalized.map(concatFileLine).join('\n')}\n',
      flush: true,
    );
    final output = path.join(temporary.path, 'full.mp4');
    final duration = probes.fold<double>(
      0,
      (sum, probe) => sum + probe.duration,
    );
    await executor.run(
      [
        '-f',
        'concat',
        '-safe',
        '0',
        '-i',
        list.path,
        '-map',
        '0:v:0',
        '-map',
        '0:a:0?',
        '-c',
        'copy',
        '-avoid_negative_ts',
        'make_zero',
        '-movflags',
        '+faststart',
        output,
      ],
      duration: duration,
      progress: (value) {
        final elapsed = value * duration;
        var boundary = 0.0;
        var index = jobs.length - 1;
        for (var i = 0; i < probes.length; i++) {
          boundary += probes[i].duration;
          if (elapsed < boundary) {
            index = i;
            break;
          }
        }
        status = '合併第 ${jobs[index].episode.number} 集中';
        progress = .7 + value * .28;
        notifyListeners();
      },
    );
    _check();
    status = '儲存合併影片';
    notifyListeners();
    // No second copy under library/: directly publish to the selected root.
    _check();
    final target = await _publishMergedMp4(
      File(output),
      jobs.first.drama,
      await repository.downloadDirectory(),
      jobs.map((job) => job.episode.number).toList(),
    );
    final relative = path.basename(target.path);
    final item = LocalMediaItem(
      id: id,
      drama: jobs.first.drama,
      file: relative,
      kind: 'merged',
      episodes: jobs.map((job) => job.episode.number).toList(),
      duration: duration,
      bytes: await target.length(),
      created: DateTime.now(),
      videoTranscodes: plan.videoTranscodes,
      audioTranscodes: plan.audioTranscodes,
      decodeVerified: false,
      sourceVersion: fingerprint,
    );
    final previous = List<LocalMediaItem>.of(_items);
    _items.removeWhere((item) => item.id == id);
    _items.add(item);
    try {
      _check();
      await _save();
    } catch (_) {
      _items = previous;
      rethrow;
    }
    return item;
  }
}
