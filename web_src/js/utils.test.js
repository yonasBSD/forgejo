import {
  basename, extname, isObject, stripTags, parseIssueHref,
  parseUrl, translateMonth, translateDay, blobToDataURI,
  toAbsoluteUrl, encodeURLEncodedBase64, decodeURLEncodedBase64, isDarkTheme, sleep, convertImage,
} from './utils.js';

test('basename', () => {
  expect(basename('/path/to/file.js')).toEqual('file.js');
  expect(basename('/path/to/file')).toEqual('file');
  expect(basename('file.js')).toEqual('file.js');
});

test('extname', () => {
  expect(extname('/path/to/file.js')).toEqual('.js');
  expect(extname('/path/')).toEqual('');
  expect(extname('/path')).toEqual('');
  expect(extname('file.js')).toEqual('.js');
});

test('isObject', () => {
  expect(isObject({})).toBeTruthy();
  expect(isObject([])).toBeFalsy();
});

test('stripTags', () => {
  expect(stripTags('<a>test</a>')).toEqual('test');
});

test('parseIssueHref', () => {
  expect(parseIssueHref('/owner/repo/issues/1')).toEqual({owner: 'owner', repo: 'repo', type: 'issues', index: '1'});
  expect(parseIssueHref('/owner/repo/pulls/1?query')).toEqual({owner: 'owner', repo: 'repo', type: 'pulls', index: '1'});
  expect(parseIssueHref('/owner/repo/issues/1#hash')).toEqual({owner: 'owner', repo: 'repo', type: 'issues', index: '1'});
  expect(parseIssueHref('/sub/owner/repo/issues/1')).toEqual({owner: 'owner', repo: 'repo', type: 'issues', index: '1'});
  expect(parseIssueHref('/sub/sub2/owner/repo/pulls/1')).toEqual({owner: 'owner', repo: 'repo', type: 'pulls', index: '1'});
  expect(parseIssueHref('/sub/sub2/owner/repo/issues/1?query')).toEqual({owner: 'owner', repo: 'repo', type: 'issues', index: '1'});
  expect(parseIssueHref('/sub/sub2/owner/repo/issues/1#hash')).toEqual({owner: 'owner', repo: 'repo', type: 'issues', index: '1'});
  expect(parseIssueHref('https://example.com/owner/repo/issues/1')).toEqual({owner: 'owner', repo: 'repo', type: 'issues', index: '1'});
  expect(parseIssueHref('https://example.com/owner/repo/pulls/1?query')).toEqual({owner: 'owner', repo: 'repo', type: 'pulls', index: '1'});
  expect(parseIssueHref('https://example.com/owner/repo/issues/1#hash')).toEqual({owner: 'owner', repo: 'repo', type: 'issues', index: '1'});
  expect(parseIssueHref('https://example.com/sub/owner/repo/issues/1')).toEqual({owner: 'owner', repo: 'repo', type: 'issues', index: '1'});
  expect(parseIssueHref('https://example.com/sub/sub2/owner/repo/pulls/1')).toEqual({owner: 'owner', repo: 'repo', type: 'pulls', index: '1'});
  expect(parseIssueHref('https://example.com/sub/sub2/owner/repo/issues/1?query')).toEqual({owner: 'owner', repo: 'repo', type: 'issues', index: '1'});
  expect(parseIssueHref('https://example.com/sub/sub2/owner/repo/issues/1#hash')).toEqual({owner: 'owner', repo: 'repo', type: 'issues', index: '1'});
  expect(parseIssueHref('')).toEqual({owner: undefined, repo: undefined, type: undefined, index: undefined});
});

test('parseUrl', () => {
  expect(parseUrl('').pathname).toEqual('/');
  expect(parseUrl('/path').pathname).toEqual('/path');
  expect(parseUrl('/path?search').pathname).toEqual('/path');
  expect(parseUrl('/path?search').search).toEqual('?search');
  expect(parseUrl('/path?search#hash').hash).toEqual('#hash');
  expect(parseUrl('https://localhost/path').pathname).toEqual('/path');
  expect(parseUrl('https://localhost/path?search').pathname).toEqual('/path');
  expect(parseUrl('https://localhost/path?search').search).toEqual('?search');
  expect(parseUrl('https://localhost/path?search#hash').hash).toEqual('#hash');
});

test('translateMonth', () => {
  const originalLang = document.documentElement.lang;
  document.documentElement.lang = 'en-US';
  expect(translateMonth(0)).toEqual('Jan');
  expect(translateMonth(4)).toEqual('May');
  document.documentElement.lang = 'es-ES';
  expect(translateMonth(5)).toEqual('jun');
  expect(translateMonth(6)).toEqual('jul');
  document.documentElement.lang = originalLang;
});

test('translateDay', () => {
  const originalLang = document.documentElement.lang;
  document.documentElement.lang = 'fr-FR';
  expect(translateDay(1)).toEqual('lun.');
  expect(translateDay(5)).toEqual('ven.');
  document.documentElement.lang = 'pl-PL';
  expect(translateDay(1)).toEqual('pon.');
  expect(translateDay(5)).toEqual('pt.');
  document.documentElement.lang = originalLang;
});

test('blobToDataURI', async () => {
  const blob = new Blob([JSON.stringify({test: true})], {type: 'application/json'});
  expect(await blobToDataURI(blob)).toEqual('data:application/json;base64,eyJ0ZXN0Ijp0cnVlfQ==');
});

test('toAbsoluteUrl', () => {
  expect(toAbsoluteUrl('//host/dir')).toEqual('http://host/dir');
  expect(toAbsoluteUrl('https://host/dir')).toEqual('https://host/dir');

  expect(toAbsoluteUrl('')).toEqual('http://localhost:3000');
  expect(toAbsoluteUrl('/user/repo')).toEqual('http://localhost:3000/user/repo');

  expect(() => toAbsoluteUrl('path')).toThrowError('unsupported');
});

test('encodeURLEncodedBase64, decodeURLEncodedBase64', () => {
  // TextEncoder is Node.js API while Uint8Array is jsdom API and their outputs are not
  // structurally comparable, so we convert to array to compare. The conversion can be
  // removed once https://github.com/jsdom/jsdom/issues/2524 is resolved.
  const encoder = new TextEncoder();
  const uint8array = encoder.encode.bind(encoder);

  expect(encodeURLEncodedBase64(uint8array('AA?'))).toEqual('QUE_'); // standard base64: "QUE/"
  expect(encodeURLEncodedBase64(uint8array('AA~'))).toEqual('QUF-'); // standard base64: "QUF+"

  expect(Array.from(decodeURLEncodedBase64('QUE/'))).toEqual(Array.from(uint8array('AA?')));
  expect(Array.from(decodeURLEncodedBase64('QUF+'))).toEqual(Array.from(uint8array('AA~')));
  expect(Array.from(decodeURLEncodedBase64('QUE_'))).toEqual(Array.from(uint8array('AA?')));
  expect(Array.from(decodeURLEncodedBase64('QUF-'))).toEqual(Array.from(uint8array('AA~')));

  expect(encodeURLEncodedBase64(uint8array('a'))).toEqual('YQ'); // standard base64: "YQ=="
  expect(Array.from(decodeURLEncodedBase64('YQ'))).toEqual(Array.from(uint8array('a')));
  expect(Array.from(decodeURLEncodedBase64('YQ=='))).toEqual(Array.from(uint8array('a')));
});

test('should sleep for a while', async () => {
  const start = Date.now();
  await sleep(100);
  const end = Date.now();
  const executionTime = end - start;
  expect(executionTime).toBeGreaterThan(90);
});

test('should return true if dark theme is enabled', () => {
  document.documentElement.style.setProperty('--is-dark-theme', 'true');
  expect(isDarkTheme()).toBe(true);
  document.documentElement.style.setProperty('--is-dark-theme', 'True ');
  expect(isDarkTheme()).toBe(true);
});

test('should return false if dark theme is disabled', () => {
  document.documentElement.style.setProperty('--is-dark-theme', 'false');
  expect(isDarkTheme()).toBe(false);
});

const data = new Uint8Array([
  137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 13, 73, 72, 68, 82, 0, 0, 0, 8,
  0, 0, 0, 8, 8, 2, 0, 0, 0, 75, 109, 41, 220, 0, 0, 0, 34, 73, 68, 65, 84,
  8, 215, 99, 120, 173, 168, 135, 21, 49, 0, 241, 255, 15, 90, 104, 8, 33,
  129, 83, 7, 97, 163, 136, 214, 129, 93, 2, 43, 2, 0, 181, 31, 90, 179,
  225, 252, 176, 37, 0, 0, 0, 0, 73, 69, 78, 68, 174, 66, 96, 130,
]);
const blob = new Blob([data], {type: 'image/png'});
test('convertImage', async () => {
  HTMLCanvasElement.prototype.getContext = vi.fn(() => {
    return {
      drawImage: vi.fn(),
    };
  });
  HTMLCanvasElement.prototype.toBlob = vi.fn((callback) => {
    callback(blob);
  });
  HTMLImageElement.prototype.addEventListener = vi.fn((_, callback) => {
    callback();
  });
  const response = await convertImage(blob, 'image/png');
  expect(response).toBe(blob);
  HTMLCanvasElement.prototype.toBlob = vi.fn((callback) => {
    callback();
  });
  try {
    await convertImage(blob, 'image/png');
  } catch (err) {
    expect(err).toBeInstanceOf(Error);
  }

  HTMLCanvasElement.prototype.toBlob = vi.fn(() => {
    throw new Error('Custom Inside Error');
  });
  try {
    await convertImage(blob, 'image/png');
  } catch (err) {
    expect(err).toBeInstanceOf(Error);
  }
  try {
    vi.spyOn(window, 'Image').mockImplementationOnce = vi.fn(() => {
      throw new Error('Custom Outside Error');
    });
    await convertImage(blob, 'image/png');
  } catch (err) {
    expect(err).toBeInstanceOf(Error);
  }
});

test('should expect error from blobToDataURI', async () => {
  try {
    vi.spyOn(window, 'FileReader').mockImplementationOnce(() => {
      this.addEventListener = vi.fn((_, callback) => callback());
      this.readAsDataURL = vi.fn(() => {
        this.result = 'data:text/plain;base64,invalid';
        this.dispatchEvent(new Event('error'));
      });
    });
    await blobToDataURI(blob);
  } catch (err) {
    expect(err).toBeInstanceOf(Error);
  }
});
