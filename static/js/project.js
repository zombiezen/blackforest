(function($) {
    $("#editbtn").click(function(e) {
        e.preventDefault();
        e.stopPropagation();

        $("#view").hide();
        $("#edit").show();
    });
    $("#editcancel").click(function(e) {
        e.preventDefault();
        e.stopPropagation();

        $("#view").show();
        $("#edit").hide();
    });
})($);
