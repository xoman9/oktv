import 'dart:async';

import 'package:duanju_app/catalog_browser.dart';
import 'package:duanju_app/models.dart';
import 'package:flutter_test/flutter_test.dart';

import 'fixtures.dart';

class StartupRepository extends FixtureRepository {
  int cacheCalls = 0;
  Completer<CatalogPage>? pending;

  @override
  Future<CatalogPage> cached(String source, {String category = ''}) {
    cacheCalls++;
    return pending?.future ?? super.cached(source, category: category);
  }

  @override
  Future<List<CatalogCategory>> categories(String source, {bool force = false}) async =>
      const [CatalogCategory.all];
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  final group = SourceGroup('hongguo', '红果', [SourceSite.byId('hongguo')]);

  test('startup displays stale library without fetching a full update', () async {
    final repository = StartupRepository();
    repository.cachedPages['hongguo'] = CatalogPage(
      [FixtureRepository.free], page: 7, hasMore: true, fresh: false,
    );
    final browser = CatalogBrowser(repository);
    final result = await browser.load(group, useCache: true);
    expect(result.items.single.id, FixtureRepository.free.id);
    expect(result.page, 7);
    expect(repository.requests, isEmpty);
    await browser.load(group, more: true);
    expect(repository.pages, [8]);
    await browser.load(group, force: true);
    expect(repository.forced.last, isTrue);
  });

  test('startup category and catalog share one pending library read', () async {
    final repository = StartupRepository()..pending = Completer<CatalogPage>();
    final browser = CatalogBrowser(repository);
    final catalog = browser.load(group, useCache: true);
    final categories = browser.loadCategories(group);
    await Future<void>.delayed(Duration.zero);
    expect(repository.cacheCalls, 1);
    repository.pending!.complete(CatalogPage([FixtureRepository.free]));
    await catalog;
    await categories;
    expect(repository.cacheCalls, 1);
    expect(browser.categories(group).any((entry) => entry.name == '合成数据'), isTrue);
  });
}
