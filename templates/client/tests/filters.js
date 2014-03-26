describe("Filters", function() {
  var datetimeFilter, rangeFilter;

  beforeEach(module('shelter'));

  beforeEach(inject(function($filter) {
    datetimeFilter = $filter("datetime");
    rangeFilter = $filter("range");
  }));

  it("should format correctly a datetime", function() {
    expect(datetimeFilter("2014-03-26T09:55:29Z", "en_US")).toBe("Wednesday, March 26 2014 9:55 AM");
    expect(datetimeFilter("2014-03-26T09:55:29Z", "pt_BR")).toBe("Quarta-feira, 26 de Março de 2014 09:55");
    expect(datetimeFilter("Invalid date", "pt_BR")).toBe("");
    expect(datetimeFilter("2014-03-26T09:55:29Z", undefined)).toBe("");
    expect(datetimeFilter("1968-01-01T09:55:29Z", "pt_BR")).toBe("");

  });
});