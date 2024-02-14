(function () {
  var CACHE_KEY = "inngest_user";
  var VERSION = "0.2.0";

  var defaults = {
    host: "inn.gs",
    await: true,
  };

  // store the key locally so as not to expose it to other JS code.
  var key = "";

  // user stores user data after a call to identify().  this is mixed
  // in with events with calls to track().
  var user = {};

  var Inngest = function () {};
  window.Inngest = Inngest;

  Inngest.options = defaults;

  /**
   * init initializes Inngest given an ingest key.  Any options provided override
   * the defaults above.
   *
   */
  Inngest.init = function (k, options) {
    var self = this;
    key = k;
    assign(self.options, options || {});
    user = get(CACHE_KEY) || {};
  };

  Inngest.event = async function (event, options) {
    var errors = validate(event, options);
    if (errors.length > 0) {
      console.warn("inngest event is invalid: ", errors.join(", "));
      return false;
    }
    event.data = event.data || {};
    assign(event.data, context());

    // The event.user object should take precedence over the identify() attributes
    // called.  Copy the event user attributes into a new variable so that we can
    // merge this into `user` from identify, then replace the original event.user.
    var overwritten = {};
    assign(overwritten, user);
    assign(overwritten, event.user || {});
    event.user = overwritten;

    var usedKey = key;
    if (options && options.key) {
      usedKey = options.key;
    }

    var body = JSON.stringify(event);
    var scheme = this.options.host === "inn.gs" ? "https://" : "//";
    var url = scheme + this.options.host + "/e/" + usedKey;

    if ((options && options.await) || this.options.await) {
      const res = await reqAsync(
        url,
        { "content-type": "application/json" },
        body
      );
      return res;
    } else {
      req(url, { "content-type": "application/json" }, body);
      return true;
    }
  };

  Inngest.track = function (eventName, data) {
    evt = {
      name: eventName,
      data: assign(data || {}, context()),
      user: user,
    };
    Inngest.event(evt);
  };

  Inngest.identify = function (externalID, data) {
    var map = {};
    if (typeof externalID === "string") {
      map.external_id = externalID;
    }
    if (typeof externalID === "object") {
      data = externalID;
    }

    // TODO: Should we merge previous identify values?
    assign(map, data);
    set(CACHE_KEY, map);

    // update local memory so that we don't have to wait on localStorage
    // and JSON serialization for each call.
    user = map;

    // XXX: We could send an "identified" event here to _guarantee_ that the
    // contact is upserted.
  };

  function set(key, val) {
    try {
      window.localStorage.setItem(key, JSON.stringify(val));
    } catch (e) {
      console.warn(e);
    }
  }

  function get(key) {
    try {
      return JSON.parse(window.localStorage.getItem(key) || "null");
    } catch (e) {
      return null;
    }
  }

  function assign(to, from) {
    iter(from, function (key, val) {
      to[key] = val;
    });
  }

  function iter(obj, callback) {
    if (typeof obj !== "object" || obj === null || obj === undefined) {
      return;
    }
    for (var o in obj) {
      if (obj.hasOwnProperty(o)) {
        callback(o, obj[o]);
      }
    }
  }

  function validate(event, options) {
    var errors = [];

    if (!key && options && !options.key) {
      errors.push(
        "init() has not been called with an ingest key, or a key was not provided"
      );
    }

    if (!event.name) {
      errors.push("event must have a name");
    }
    return errors;
  }

  function context() {
    var data = {
      context: {
        path: window.location.pathname,
        url: window && window.location.href,
        title: document && document.title,
        search: window && window.location.search,
        referrer: document && document.referrer,
        user_agent: navigator && navigator.userAgent,
        library: "js",
        library_version: VERSION,
        // TODO Store utm params
      },
    };
    if (window && window.location.search.length) {
      try {
        var params = new URLSearchParams(window.location.search);
        ["source", "medium", "campaign", "content", "term"].forEach(function (
          param
        ) {
          var key = "utm_" + param;
          if (params.get(key)) {
            data.context[key] = params.get(key);
          }
        });
      } catch (err) {
        /* No-op - URLSearchParams may not be supported in browser */
      }
    }
    return data;
  }

  function req(url, headers, body) {
    var r = new XMLHttpRequest();
    r.open("POST", url);
    r.withCredentials = false;

    headers = headers || {};
    for (var header in headers) {
      if (headers.hasOwnProperty(header)) {
        r.setRequestHeader(header, headers[header]);
      }
    }

    r.send(body);
  }

  function reqAsync(url, headers, body) {
    return new Promise(function (resolve, _reject) {
      var r = new XMLHttpRequest();
      r.open("POST", url);
      r.withCredentials = false;

      headers = headers || {};
      for (var header in headers) {
        if (headers.hasOwnProperty(header)) {
          r.setRequestHeader(header, headers[header]);
        }
      }

      r.onreadystatechange = function () {
        if (r.readyState === XMLHttpRequest.DONE) {
          const status = r.status;
          const error =
            status >= 200 && status < 400
              ? null
              : "There was an error with this request.";
          resolve({
            status,
            response: r.response,
            error,
          });
        }
      };

      r.send(body);
    });
  }
})();
