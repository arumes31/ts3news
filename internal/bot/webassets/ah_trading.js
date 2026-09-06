(function () {
  'use strict';

  var tradingTabs = document.querySelector('.auction-trading-tabs');
  window.showAHTrading = function (kind) {
    document.querySelectorAll('[data-ah-trading]').forEach(function (panel) {
      var selected = panel.dataset.ahTrading === kind;
      panel.hidden = !selected;
      var tab = document.getElementById(panel.getAttribute('aria-labelledby'));
      tab.setAttribute('aria-selected', String(selected));
      tab.tabIndex = selected ? 0 : -1;
      tab.classList.toggle('active', selected);
    });
  };
  tradingTabs.addEventListener('keydown', function (event) {
    var tabs = Array.from(tradingTabs.querySelectorAll('[role="tab"]'));
    var index = tabs.indexOf(document.activeElement);
    if (index < 0 || !['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
    event.preventDefault();
    var next = event.key === 'Home' ? 0 : event.key === 'End' ? tabs.length - 1
      : (index + (event.key === 'ArrowRight' ? 1 : -1) + tabs.length) % tabs.length;
    tabs[next].click();
    tabs[next].focus();
  });

  var noticePage = [];
  var noticeBefore = 0;
  var noticeNext = 0;
  var noticeLoading = false;
  var noticeStatus = document.getElementById('ahNoticeStatus');
  var noticeRetry = document.getElementById('ahNoticeRetry');
  function noticeMessage(message, error) {
    noticeStatus.setAttribute('role', error ? 'alert' : 'status');
    noticeStatus.textContent = message;
    noticeStatus.classList.toggle('auction-field-error', !!error);
  }
  function noticeControls() {
    document.getElementById('ahNoticeRead').disabled = noticeLoading || !noticePage.some(function (notice) { return !notice.seen; });
    document.getElementById('ahNoticeOlder').disabled = noticeLoading;
    document.getElementById('ahNoticeNewest').disabled = noticeLoading;
    noticeRetry.disabled = noticeLoading;
  }
  function renderNotices() {
    var list = document.getElementById('ahNoticeList');
    list.replaceChildren();
    noticePage.forEach(function (notice) {
      var item = document.createElement('li');
      var meta = document.createElement('div');
      meta.className = 'auction-notice-meta';
      var time = document.createElement('time');
      time.textContent = notice.when;
      if (notice.when_iso) time.dateTime = notice.when_iso;
      meta.appendChild(time);
      if (!notice.seen) {
        var unread = document.createElement('strong');
        unread.textContent = 'Unread';
        meta.appendChild(unread);
      }
      var message = document.createElement('p');
      message.textContent = notice.message;
      item.append(meta, message);
      var destination = notice.kind === 'order_fill' ? 'orders'
        : ['sale', 'bid_win', 'bid_refund', 'outbid'].includes(notice.kind) ? 'transactions' : '';
      if (destination || ['watchlist', 'price_alert'].includes(notice.kind)) {
        var link = document.createElement('a');
        link.href = destination ? '#myTrading' : '#auctionListingsTitle';
        link.textContent = destination === 'orders' ? 'View orders' : destination ? 'View transactions' : 'Browse listings';
        if (destination) link.addEventListener('click', function () { showAHTrading(destination); });
        item.appendChild(link);
      }
      list.appendChild(item);
    });
  }
  window.toggleAHNotices = function (button) {
    var panel = document.getElementById('ahNotices');
    panel.hidden = !panel.hidden;
    button.setAttribute('aria-expanded', String(!panel.hidden));
    if (!panel.hidden) {
      document.getElementById('ahNoticesTitle').focus({preventScroll: true});
      loadAHNoticePage(0);
    }
  };
  window.loadAHNoticePage = async function (before) {
    if (noticeLoading) return;
    noticeBefore = before;
    noticeLoading = true;
    noticeRetry.hidden = true;
    noticeMessage('Loading market notices…');
    noticeControls();
    var failureMessage = 'Could not load market notices. Retry to view your saved activity.';
    try {
      var response = await fetch('/api/ah/notices' + (before ? '?before=' + encodeURIComponent(before) : ''), {cache: 'no-store'});
      var data = await response.json();
      if (!response.ok || !data.ok || !Array.isArray(data.notices)) {
        if (typeof data.error === 'string') failureMessage = data.error + ' Retry to view your saved activity.';
        throw new Error('Notice response failed');
      }
      noticePage = data.notices;
      noticeNext = data.has_more ? data.next_before : 0;
      renderNotices();
      document.getElementById('ahNoticeCount').textContent = data.unseen_count;
      document.getElementById('ahNoticeOlder').hidden = !noticeNext;
      document.getElementById('ahNoticeNewest').hidden = !before;
      noticeMessage(noticePage.length ? 'Notices remain available after you mark them as read.' : 'No market notices yet. Your trading updates will appear here.');
    } catch (error) {
      noticeMessage(failureMessage, true);
      noticeRetry.hidden = false;
    } finally {
      noticeLoading = false;
      noticeControls();
    }
  };
  window.retryAHNotices = function () { loadAHNoticePage(noticeBefore); };
  window.loadOlderAHNotices = function () { if (noticeNext) loadAHNoticePage(noticeNext); };
  window.markAHNoticesRead = async function () {
    if (noticeLoading) return;
    var ids = noticePage.filter(function (notice) { return !notice.seen; }).map(function (notice) { return notice.id; });
    if (!ids.length) return;
    noticeLoading = true;
    noticeControls();
    var data = await post('/api/ah/notices', {ids: ids});
    if (data.ok) {
      noticePage.forEach(function (notice) { notice.seen = true; });
      renderNotices();
      document.getElementById('ahNoticeCount').textContent = data.unseen_count;
      noticeMessage('Marked shown notices as read. They remain in your history.');
    } else {
      noticeMessage('Could not confirm the read status. Your notices are still here; retry marking them as read.', true);
    }
    noticeLoading = false;
    noticeControls();
  };

  var orderForm = document.getElementById('ahOrderForm');
  var orderQuantity = document.getElementById('ahOrderQuantity');
  var orderUnit = document.getElementById('ahOrderUnit');
  var orderStart = document.getElementById('ahOrderStart');
  var orderStatus = document.getElementById('ahOrderStatus');
  var orderBusy = false;
  var orderLocked = false;
  function wholeNumber(input) {
    var value = input.value.trim();
    return /^\d+$/.test(value) && Number.isSafeInteger(Number(value)) ? Number(value) : NaN;
  }
  function orderValues(validate) {
    var count = wholeNumber(orderQuantity);
    var unit = wholeNumber(orderUnit);
    var quantityError = !Number.isInteger(count) || count < 1 || count > 10000 ? 'Enter a whole quantity from 1 to 10,000.' : '';
    var unitError = !Number.isInteger(unit) || unit < 1 ? 'Enter a positive whole-gold price.' : '';
    var escrow = count * unit;
    if (!quantityError && !unitError && !Number.isSafeInteger(escrow)) unitError = 'This escrow total is too large to review safely. Lower the quantity or price.';
    if (validate) {
      document.getElementById('ahOrderQuantityError').textContent = quantityError;
      document.getElementById('ahOrderUnitError').textContent = unitError;
      orderQuantity.setAttribute('aria-invalid', String(!!quantityError));
      orderUnit.setAttribute('aria-invalid', String(!!unitError));
    }
    var valid = !quantityError && !unitError;
    document.getElementById('ahOrderEscrow').textContent = valid ? 'Escrow: ' + escrow.toLocaleString() + ' gold' : 'Escrow: enter valid whole numbers';
    return {valid: valid, count: count, unit: unit, escrow: escrow, firstInvalid: quantityError ? orderQuantity : orderUnit};
  }
  function setOrderBusy(busy) {
    orderBusy = busy;
    document.getElementById('ahOrderFields').disabled = busy;
    document.getElementById('ahOrderReview').disabled = busy;
    document.getElementById('ahOrderClose').disabled = busy;
    orderStart.disabled = busy;
    orderForm.setAttribute('aria-busy', String(busy));
  }
  orderForm.addEventListener('input', function () { orderValues(orderForm.dataset.validated === 'true'); });
  window.createMaterialOrder = function () {
    if (orderBusy || orderLocked) return;
    orderForm.hidden = false;
    orderValues(false);
    document.getElementById('ahOrderMaterial').focus();
  };
  window.closeAHMaterialOrder = function () {
    if (orderBusy || orderLocked) return;
    orderForm.hidden = true;
    orderStart.focus();
  };
  window.submitAHMaterialOrder = async function (event) {
    event.preventDefault();
    if (orderBusy || orderLocked) return;
    orderForm.dataset.validated = 'true';
    orderStatus.setAttribute('role', 'status');
    orderStatus.textContent = '';
    var values = orderValues(true);
    if (!values.valid) { values.firstInvalid.focus(); return; }
    var material = document.getElementById('ahOrderMaterial').value;
    var review = document.getElementById('ahOrderReview');
    setOrderBusy(true);
    var confirmed = await confirmAH('Post an order for ' + values.count.toLocaleString() + ' ' + ahEscapeHTML(material) + ' at ' + values.unit.toLocaleString() + ' gold each?<br>Exact escrow: ' + values.escrow.toLocaleString() + ' gold · posting fee: 0 gold.', {
      title: 'Confirm buy order', okLabel: 'Reserve ' + values.escrow.toLocaleString() + ' gold', cancelLabel: 'Edit order', returnFocus: review
    });
    if (!confirmed) { setOrderBusy(false); review.focus(); return; }
    orderStatus.textContent = 'Posting your buy order…';
    var data = await post('/api/ah/material_order', {material: material, count: values.count, unit_price: values.unit});
    if (data.ok) {
      orderStatus.textContent = data.msg || 'Buy order posted.';
      toast(orderStatus.textContent, true);
      setTimeout(function () { location.reload(); }, 700);
      return;
    }
    orderStatus.setAttribute('role', 'alert');
    orderStatus.textContent = data.error || 'Your order could not be posted. Review the details and try again.';
    if (data.unconfirmed) {
      orderLocked = true;
      orderForm.setAttribute('aria-busy', 'false');
      failAHMutation(data, review);
    } else {
      setOrderBusy(false);
      review.focus();
    }
  };

  window.addAuctionInspectorContext = function (trigger) {
    var row = trigger.closest('.auction-listings .ah-row');
    if (!row) return;
    var inspector = document.querySelector('#sharedModal .item-inspector');
    var buy = row.querySelector('.auction-buy-button');
    var context = document.createElement('section');
    context.className = 'auction-inspector-market';
    var heading = document.createElement('h3');
    heading.textContent = 'Auction listing';
    var price = document.createElement('p');
    price.className = 'auction-inspector-price';
    price.textContent = 'Buy now: ' + Number(row.dataset.ahPrice).toLocaleString() + ' gold · buyer fee: 0 gold';
    var delivery = document.createElement('p');
    delivery.textContent = buy ? ahDeliveryDescription(buy) : 'Your listing is available to other players.';
    context.append(heading, price, delivery);
    if (buy) {
      var review = document.createElement('button');
      review.type = 'button';
      review.textContent = 'Review purchase';
      review.disabled = buy.disabled;
      review.addEventListener('click', function () {
        closeModal();
        buyAH(row.dataset.ahId, buy, trigger);
      });
      context.appendChild(review);
    }
    inspector.querySelector('.item-inspector-head').after(context);
  };
}());
