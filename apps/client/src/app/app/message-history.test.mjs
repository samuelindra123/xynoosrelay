import { test } from 'node:test';
import { strict as assert } from 'node:assert';
import { refreshHistory, prependHistory } from './message-history.ts';

test('polling preserves older pages and replaces the recent window', () => {
  assert.deepEqual(refreshHistory([{id:1},{id:2},{id:3}], [{id:2,text:'changed'},{id:4}]), [{id:1},{id:2,text:'changed'},{id:4}]);
  assert.deepEqual(refreshHistory([{id:1}], []), []);
});
test('overlapping history pages deduplicate and preserve newer data', () => {
  assert.deepEqual(prependHistory([{id:3,text:'new'},{id:4}], [{id:1},{id:3,text:'old'}]), [{id:1},{id:3,text:'new'},{id:4}]);
});
